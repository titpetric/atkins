package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// writeSkill writes a skill file, and its markdown companion when doc is
// not empty, into a skills directory.
func writeSkill(t *testing.T, dir, id, yaml, doc string) {
	t.Helper()

	assert.NoError(t, os.MkdirAll(dir, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, id+".yml"), []byte(yaml), 0o644))

	if doc != "" {
		assert.NoError(t, os.WriteFile(filepath.Join(dir, id+".md"), []byte(doc), 0o644))
	}
}

// TestSkillsLoader_Scan checks Scan returns inactive skills too, reads
// the markdown companion beside a skill file, and keeps the help: line.
func TestSkillsLoader_Scan(t *testing.T) {
	root := t.TempDir()
	skillsDir := filepath.Join(root, ".atkins", "skills")

	writeSkill(t, skillsDir, "go", `
name: Go build and test
help: Builds and tests a Go module.
when:
  files:
    - go.mod
jobs:
  build: go build ./...
`, "\n`atkins go:build` builds the module.\n")

	writeSkill(t, skillsDir, "mdox", `
name: Markdown
jobs:
  fmt: mdox fmt README.md
`, "")

	loader := runner.NewSkillsLoader(root, root)

	skills, err := loader.Scan()
	assert.NoError(t, err)
	assert.Len(t, skills, 2)

	byID := map[string]*runner.Skill{}
	for _, skill := range skills {
		byID[skill.ID()] = skill
	}

	goSkill := byID["go"]
	assert.NotNil(t, goSkill)
	assert.False(t, goSkill.Active, "no go.mod above the temporary directory")
	assert.Equal(t, "Builds and tests a Go module.", goSkill.Pipeline.Help)
	assert.Equal(t, filepath.Join(skillsDir, "go.md"), goSkill.DocPath)
	assert.Equal(t, "`atkins go:build` builds the module.", goSkill.Doc)

	mdox := byID["mdox"]
	assert.NotNil(t, mdox)
	assert.True(t, mdox.Active, "a skill without when: is active everywhere")
	assert.Empty(t, mdox.DocPath)
	assert.Empty(t, mdox.Doc)
}

// TestSkillsLoader_ScanKeepsDirectoryOrder checks the same skill ID from
// two directories appears twice, the higher-priority directory first, so
// a caller can report precedence.
func TestSkillsLoader_ScanKeepsDirectoryOrder(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, ".atkins", "skills")
	global := filepath.Join(root, "global", "skills")

	writeSkill(t, local, "go", "name: Local\njobs:\n  build: go build ./...\n", "")
	writeSkill(t, global, "go", "name: Global\njobs:\n  build: go build ./...\n", "")

	loader := runner.NewSkillsLoader(root, root)
	loader.AddSkillsDir(global)

	skills, err := loader.Scan()
	assert.NoError(t, err)
	assert.Len(t, skills, 2)
	assert.Equal(t, "Local", skills[0].Pipeline.Name)
	assert.Equal(t, "Global", skills[1].Pipeline.Name)
}

// TestSkillsLoader_Load checks Load drops the skills that do not apply
// here and keeps the first active one per ID.
func TestSkillsLoader_Load(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, ".atkins", "skills")
	global := filepath.Join(root, "global", "skills")

	writeSkill(t, local, "go", "name: Local\njobs:\n  build: go build ./...\n", "")
	writeSkill(t, global, "go", "name: Global\njobs:\n  build: go build ./...\n", "")
	writeSkill(t, global, "tailwind", `
name: Tailwind
when:
  files:
    - tailwind.config.js
jobs:
  generate: tailwindcss
`, "")

	loader := runner.NewSkillsLoader(root, root)
	loader.AddSkillsDir(global)

	pipelines, err := loader.Load()
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "Local", pipelines[0].Name)
	assert.Equal(t, root, pipelines[0].Dir)
}

// TestSkillsLoader_LoadSkipsShadowedInactive checks an inactive skill
// does not shadow the active one of the same ID in a later directory.
func TestSkillsLoader_LoadSkipsShadowedInactive(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, ".atkins", "skills")
	global := filepath.Join(root, "global", "skills")

	writeSkill(t, local, "go", `
name: Local
when:
  files:
    - never-present.txt
jobs:
  build: go build ./...
`, "")
	writeSkill(t, global, "go", "name: Global\njobs:\n  build: go build ./...\n", "")

	loader := runner.NewSkillsLoader(root, root)
	loader.AddSkillsDir(global)

	pipelines, err := loader.Load()
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "Global", pipelines[0].Name)
}

// TestParentWorkspaces checks the walk lists the folders above a project
// holding .atkins/skills, nearest first, and stops below home.
func TestParentWorkspaces(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	ws := filepath.Join(home, "ws")
	group := filepath.Join(ws, "group")
	proj := filepath.Join(group, "proj")

	writeSkill(t, filepath.Join(root, ".atkins", "skills"), "above", "jobs:\n  a: true\n", "")
	writeSkill(t, filepath.Join(home, ".atkins", "skills"), "global", "jobs:\n  a: true\n", "")
	writeSkill(t, filepath.Join(ws, ".atkins", "skills"), "ws", "jobs:\n  a: true\n", "")
	writeSkill(t, filepath.Join(group, ".atkins", "skills"), "group", "jobs:\n  a: true\n", "")
	assert.NoError(t, os.MkdirAll(proj, 0o755))

	assert.Equal(t, []string{group, ws}, runner.ParentWorkspaces(proj, home))
	assert.Empty(t, runner.ParentWorkspaces(home, home))

	// Outside home the walk reaches the filesystem root.
	assert.Equal(t, []string{group, ws, home, root}, runner.ParentWorkspaces(proj, filepath.Join(root, "elsewhere"))[:4])
}

// TestSkillsLoader_ParentWorkspace checks a project inside a workspace
// checkout loads the workspace's skills, runs one without when: in the
// workspace, and lets its own skill of the same ID win.
func TestSkillsLoader_ParentWorkspace(t *testing.T) {
	ws := t.TempDir()
	proj := filepath.Join(ws, "proj")

	writeSkill(t, filepath.Join(ws, ".atkins", "skills"), "a", "name: Workspace a\njobs:\n  default: pwd\n", "")
	writeSkill(t, filepath.Join(ws, ".atkins", "skills"), "shared", "name: Workspace shared\njobs:\n  default: pwd\n", "")
	writeSkill(t, filepath.Join(proj, ".atkins", "skills"), "shared", "name: Project shared\njobs:\n  default: pwd\n", "")
	assert.NoError(t, os.WriteFile(filepath.Join(proj, "atkins.yml"), []byte("jobs:\n  default: true\n"), 0o644))

	loader := runner.NewSkillsLoader(proj, proj)
	loader.AddParentWorkspaces()

	pipelines, err := loader.Load()
	assert.NoError(t, err)

	byID := map[string]string{}
	dirs := map[string]string{}
	for _, p := range pipelines {
		byID[p.ID] = p.Name
		dirs[p.ID] = p.Dir
	}

	assert.Equal(t, "Workspace a", byID["a"])
	assert.Equal(t, ws, dirs["a"])
	assert.Equal(t, "Project shared", byID["shared"])
	assert.Equal(t, proj, dirs["shared"])
}

// TestSkillsLoader_AddWorkspace checks a skill without when: from an
// added workspace runs in that workspace, not in WorkspaceDir.
func TestSkillsLoader_AddWorkspace(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(root, "other")

	writeSkill(t, filepath.Join(other, ".atkins", "skills"), "lint", "jobs:\n  default: true\n", "")

	loader := runner.NewSkillsLoader(root, root)
	loader.AddWorkspace(other)

	pipelines, err := loader.Load()
	assert.NoError(t, err)
	assert.Len(t, pipelines, 1)
	assert.Equal(t, other, pipelines[0].Dir)
}

// TestSkillsLoader_LoadAll checks LoadAll returns the inactive skills
// after the active ones, marked Inactive, and that a skill from a
// parent folder reports its path and the folder it runs in.
func TestSkillsLoader_LoadAll(t *testing.T) {
	t.Parallel()

	ws := t.TempDir()
	pulse := filepath.Join(ws, "pulse")
	assert.NoError(t, os.MkdirAll(pulse, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(pulse, "compose.yml"), []byte("services: {}\n"), 0o644))

	skillsDir := filepath.Join(ws, ".atkins", "skills")
	writeSkill(t, skillsDir, "compose", "when:\n  files: [compose.yml]\n  here: true\njobs:\n  up: docker compose up -d\n", "")
	writeSkill(t, skillsDir, "composer", "when:\n  files: [composer.lock]\n  here: true\njobs:\n  update: composer update\n", "")

	loader := runner.NewSkillsLoader(pulse, pulse)
	loader.AddWorkspace(ws)

	active, err := loader.Load()
	assert.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "compose", active[0].ID)

	all, err := loader.LoadAll()
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	compose, composer := all[0], all[1]
	assert.Equal(t, "compose", compose.ID)
	assert.False(t, compose.Inactive)
	assert.Equal(t, filepath.Join(skillsDir, "compose.yml"), compose.Path)
	assert.Equal(t, pulse, compose.Dir)
	assert.Equal(t, filepath.Join(pulse, "compose.yml"), compose.Matched)

	assert.Equal(t, "composer", composer.ID)
	assert.True(t, composer.Inactive)
	assert.Equal(t, "", composer.Dir)
}

// TestMergeSkills checks MergeSkills skips an ID already present,
// unless the present one is inactive and the new one applies.
func TestMergeSkills(t *testing.T) {
	t.Parallel()

	main := &model.Pipeline{Name: "main"}
	localGo := &model.Pipeline{ID: "go"}
	localNode := &model.Pipeline{ID: "node", Inactive: true}
	globalGo := &model.Pipeline{ID: "go"}
	globalNode := &model.Pipeline{ID: "node"}
	globalRust := &model.Pipeline{ID: "rust", Inactive: true}

	got := runner.MergeSkills(
		[]*model.Pipeline{main, localGo, localNode},
		[]*model.Pipeline{globalGo, globalNode, globalRust},
	)

	assert.Len(t, got, 4)
	assert.True(t, got[0] == main)
	assert.True(t, got[1] == localGo, "the local go skill shadows the global one")
	assert.True(t, got[2] == globalNode, "an active skill replaces an inactive one")
	assert.True(t, got[3] == globalRust)
}
