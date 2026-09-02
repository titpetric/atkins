package runner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

// writeFile creates a file and any parent directories it needs.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}

// vendorFixture creates a repository and a source skills directory,
// returning both. The repository has a .git folder so it looks like a
// checkout to FindRepositoryRoot.
func vendorFixture(t *testing.T) (root, source string) {
	t.Helper()

	base := t.TempDir()
	root = filepath.Join(base, "repo")
	source = filepath.Join(base, "home", ".atkins", "skills")

	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(source, 0o755))

	return root, source
}

// vendorIDs lists the skill IDs of a result set.
func vendorIDs(skills []*runner.VendorSkill) []string {
	result := make([]string, 0, len(skills))
	for _, skill := range skills {
		result = append(result, skill.ID)
	}
	return result
}

const goSkill = `name: Go build and test
when:
  files:
    - go.mod
jobs:
  build:
    steps:
      - run: go build ./...
`

const dockerSkill = `name: Docker build
when:
  files:
    - docker/Dockerfile
jobs:
  build:
    steps:
      - run: docker build .
`

const migrationSkill = `name: SQL Migrations
when:
  files:
    - schema/*.up.sql
jobs:
  migrate:
    steps:
      - run: mig migrate
`

// TestFindRepositoryRoot covers locating the folder that holds .git.
func TestFindRepositoryRoot(t *testing.T) {
	t.Run("finds root from a subdirectory", func(t *testing.T) {
		root, _ := vendorFixture(t)
		sub := filepath.Join(root, "server", "web")
		require.NoError(t, os.MkdirAll(sub, 0o755))

		found, err := runner.FindRepositoryRoot(sub)
		require.NoError(t, err)
		assert.Equal(t, root, found)
	})

	t.Run("finds root when .git is a file", func(t *testing.T) {
		base := t.TempDir()
		writeFile(t, filepath.Join(base, ".git"), "gitdir: /elsewhere/.git/worktrees/x\n")

		found, err := runner.FindRepositoryRoot(base)
		require.NoError(t, err)
		assert.Equal(t, base, found)
	})

	t.Run("errors outside a repository", func(t *testing.T) {
		_, err := runner.FindRepositoryRoot(t.TempDir())
		require.Error(t, err)
	})
}

// TestVendorerPlan covers which skills a repository is found to use.
func TestVendorerPlan(t *testing.T) {
	t.Run("matches when: files in the repository root", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(source, "docker.yml"), dockerSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"docker", "go"}, vendorIDs(result.Found))
		assert.Equal(t, []string{"go"}, vendorIDs(result.Used))
		assert.Equal(t, "go.mod", result.Used[0].Reason)
	})

	t.Run("matches when: files in a subdirectory", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "schema.yml"), "when:\n  files:\n    - schema/\njobs:\n  migrate:\n    steps:\n      - run: mig migrate\n")
		require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "users", "schema"), 0o755))

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"schema"}, vendorIDs(result.Used))
		assert.Equal(t, filepath.Join("modules", "users", "schema"), result.Used[0].Reason)
	})

	t.Run("matches a glob in when: files", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "schema.yml"), migrationSkill)
		writeFile(t, filepath.Join(root, "server", "schema", "user.up.sql"), "CREATE TABLE user (id INT);\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"schema"}, vendorIDs(result.Used))
		assert.Equal(t, filepath.Join("server", "schema", "user.up.sql"), result.Used[0].Reason)
	})

	t.Run("leaves a glob in when: files unmatched by an empty folder", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "schema.yml"), migrationSkill)
		require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "schema"), 0o755))
		writeFile(t, filepath.Join(root, "docs", "schema", "jobs.md"), "# Jobs\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Empty(t, result.Used)
	})

	t.Run("skips directories the walk excludes", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "runner", "testdata", "go.mod"), "module fixture\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Empty(t, result.Used)
	})

	t.Run("takes a skill without when: as always active", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "notes.yml"), "jobs:\n  default:\n    steps:\n      - run: echo hi\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"notes"}, vendorIDs(result.Used))
		assert.Equal(t, "always active", result.Used[0].Reason)
	})

	t.Run("takes a skill a pipeline references", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "docker.yml"), dockerSkill)
		writeFile(t, filepath.Join(root, "atkins.yml"), "name: Example\njobs:\n  release:\n    steps:\n      - task: docker:build\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"docker"}, vendorIDs(result.Used))
		assert.Equal(t, "referenced by atkins.yml", result.Used[0].Reason)
	})

	t.Run("takes a skill a nested pipeline depends on", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "docker.yml"), dockerSkill)
		writeFile(t, filepath.Join(root, "app", ".atkins.yml"), "name: App\njobs:\n  ship:\n    depends_on: [\"docker:build\"]\n    steps:\n      - run: echo ship\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"docker"}, vendorIDs(result.Used))
	})

	t.Run("follows references between skills", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "docker.yml"), dockerSkill)
		writeFile(t, filepath.Join(source, "release.yml"), "when:\n  files:\n    - .github/\njobs:\n  default:\n    steps:\n      - task: :docker:build\n")
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".github"), 0o755))

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Equal(t, []string{"docker", "release"}, vendorIDs(result.Used))
		assert.Equal(t, "referenced by release", result.Used[0].Reason)
	})

	t.Run("ignores a task that names a job of its own pipeline", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "build.yml"), "jobs:\n  default:\n    steps:\n      - run: echo hi\n")
		writeFile(t, filepath.Join(root, "atkins.yml"), "name: Example\njobs:\n  build:\n    steps:\n      - run: echo build\n  default:\n    steps:\n      - task: build\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		// The skill is still used, but because it has no when: block,
		// not because the pipeline's own `build` job was mistaken for it.
		require.Equal(t, []string{"build"}, vendorIDs(result.Used))
		assert.Equal(t, "always active", result.Used[0].Reason)
	})

	t.Run("reports an empty source directory", func(t *testing.T) {
		root, _ := vendorFixture(t)

		result, err := runner.NewVendorer(filepath.Join(root, "missing"), root).Plan()
		require.NoError(t, err)
		assert.Empty(t, result.Found)
		assert.Empty(t, result.Used)
	})

	t.Run("refuses to vendor onto itself", func(t *testing.T) {
		root, _ := vendorFixture(t)
		source := filepath.Join(root, ".atkins", "skills")

		_, err := runner.NewVendorer(source, root).Plan()
		require.Error(t, err)
	})
}

// TestVendorerPlanDiff covers the status and line counts a dry run reports.
func TestVendorerPlanDiff(t *testing.T) {
	t.Run("counts a new skill as all additions", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		require.Len(t, result.Used, 1)

		skill := result.Used[0]
		assert.Equal(t, runner.VendorNew, skill.Status)
		assert.Equal(t, 8, skill.Added)
		assert.Equal(t, 0, skill.Removed)
		assert.Empty(t, skill.Diff)
		assert.Equal(t, []string{"go"}, result.Pending())
	})

	t.Run("reports an identical copy as up to date", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, ".atkins", "skills", "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		require.Len(t, result.Used, 1)

		skill := result.Used[0]
		assert.Equal(t, runner.VendorCurrent, skill.Status)
		assert.Zero(t, skill.Added)
		assert.Zero(t, skill.Removed)
		assert.Empty(t, skill.Diff)
		assert.Empty(t, result.Pending())
	})

	t.Run("counts the lines a changed copy differs by", func(t *testing.T) {
		root, source := vendorFixture(t)
		local := strings.Replace(goSkill, "      - run: go build ./...\n", "      - run: go build ./cmd/...\n      - run: go vet ./...\n", 1)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, ".atkins", "skills", "go.yml"), local)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		require.Len(t, result.Used, 1)

		skill := result.Used[0]
		assert.Equal(t, runner.VendorChanged, skill.Status)
		assert.Equal(t, 1, skill.Added)
		assert.Equal(t, 2, skill.Removed)
		assert.Contains(t, skill.Diff, "+      - run: go build ./...")
		assert.Contains(t, skill.Diff, "-      - run: go vet ./...")
		assert.Contains(t, skill.Diff, filepath.Join(".atkins", "skills", "go.yml"))
	})

	t.Run("writes nothing", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		assert.Empty(t, result.Installed)

		_, err = os.Stat(filepath.Join(root, ".atkins"))
		assert.True(t, os.IsNotExist(err))
	})
}

// TestVendorerRun covers writing the selected skills into the repository.
func TestVendorerRun(t *testing.T) {
	t.Run("creates .atkins/skills and copies the file", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Equal(t, []string{"go"}, result.Installed)

		vendored, err := os.ReadFile(filepath.Join(root, ".atkins", "skills", "go.yml"))
		require.NoError(t, err)
		assert.Equal(t, goSkill, string(vendored))
	})

	t.Run("writes nothing on a second run", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		_, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Empty(t, result.Installed)
		assert.Empty(t, result.Pending())
	})

	t.Run("overwrites a vendored copy that has drifted", func(t *testing.T) {
		root, source := vendorFixture(t)
		local := "name: Project go\nwhen:\n  files:\n    - go.mod\njobs:\n  build:\n    steps:\n      - run: go build ./cmd/...\n"
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, ".atkins", "skills", "go.yml"), local)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Equal(t, []string{"go"}, result.Installed)

		vendored, err := os.ReadFile(filepath.Join(root, ".atkins", "skills", "go.yml"))
		require.NoError(t, err)
		assert.Equal(t, goSkill, string(vendored))
	})

	t.Run("writes nothing when no skill is used", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Empty(t, result.Installed)

		_, err = os.Stat(filepath.Join(root, ".atkins"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("takes a skill a vendored skill references", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "docker.yml"), dockerSkill)
		writeFile(t, filepath.Join(root, ".atkins", "skills", "deploy.yml"), "jobs:\n  default:\n    steps:\n      - task: docker:build\n")

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Equal(t, []string{"docker"}, result.Installed)
	})
}

// TestVendorerRunCompanion covers the markdown companion beside a skill
// file: it is installed with the skill, counted in the diff, and a
// change to it alone is enough to make the skill out of date.
func TestVendorerRunCompanion(t *testing.T) {
	const guide = "`atkins go:build` builds the module.\n"

	t.Run("installs the companion with the skill", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(source, "go.md"), guide)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		result, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)
		assert.Equal(t, []string{"go"}, result.Installed)

		vendored, err := os.ReadFile(filepath.Join(root, ".atkins", "skills", "go.md"))
		require.NoError(t, err)
		assert.Equal(t, guide, string(vendored))
	})

	t.Run("reports a skill whose companion drifted", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(source, "go.md"), guide)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")
		writeFile(t, filepath.Join(root, ".atkins", "skills", "go.yml"), goSkill)
		writeFile(t, filepath.Join(root, ".atkins", "skills", "go.md"), "stale\n")

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		require.Len(t, result.Used, 1)
		assert.Equal(t, runner.VendorChanged, result.Used[0].Status)
		assert.Equal(t, []string{"go"}, result.Pending())
		assert.Contains(t, result.Used[0].Diff, "stale")
	})

	t.Run("leaves a skill current when both files match", func(t *testing.T) {
		root, source := vendorFixture(t)
		writeFile(t, filepath.Join(source, "go.yml"), goSkill)
		writeFile(t, filepath.Join(source, "go.md"), guide)
		writeFile(t, filepath.Join(root, "go.mod"), "module example\n")

		_, err := runner.NewVendorer(source, root).Run()
		require.NoError(t, err)

		result, err := runner.NewVendorer(source, root).Plan()
		require.NoError(t, err)
		require.Len(t, result.Used, 1)
		assert.Equal(t, runner.VendorCurrent, result.Used[0].Status)
		assert.Empty(t, result.Pending())
	})
}
