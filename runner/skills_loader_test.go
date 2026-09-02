package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

// writeSkill writes a skill file, and its markdown companion when doc is
// not empty, into a skills directory.
func writeSkill(t *testing.T, dir, id, yaml, doc string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, id+".yml"), []byte(yaml), 0o644))

	if doc != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, id+".md"), []byte(doc), 0o644))
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
	require.NoError(t, err)
	require.Len(t, skills, 2)

	byID := map[string]*runner.Skill{}
	for _, skill := range skills {
		byID[skill.ID()] = skill
	}

	goSkill := byID["go"]
	require.NotNil(t, goSkill)
	assert.False(t, goSkill.Active, "no go.mod above the temporary directory")
	assert.Equal(t, "Builds and tests a Go module.", goSkill.Pipeline.Help)
	assert.Equal(t, filepath.Join(skillsDir, "go.md"), goSkill.DocPath)
	assert.Equal(t, "`atkins go:build` builds the module.", goSkill.Doc)

	mdox := byID["mdox"]
	require.NotNil(t, mdox)
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
	require.NoError(t, err)
	require.Len(t, skills, 2)
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
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
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
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	assert.Equal(t, "Global", pipelines[0].Name)
}
