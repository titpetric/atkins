package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/titpetric/atkins/runner"
)

// TestFindFolder tests the FindFolder method for locating directories
// by name, traversing parent directories.
func TestFindFolder(t *testing.T) {
	t.Run("finds folder in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		atkinsDir := filepath.Join(tmpDir, ".atkins")
		require.NoError(t, os.MkdirAll(atkinsDir, 0o755))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		dir, found := loader.FindFolder(".atkins", tmpDir)
		assert.True(t, found)
		assert.Equal(t, tmpDir, dir)
	})

	t.Run("finds folder in parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub", "folder")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		atkinsDir := filepath.Join(tmpDir, ".atkins")
		require.NoError(t, os.MkdirAll(atkinsDir, 0o755))

		loader := runner.NewSkillsLoader(tmpDir, subDir)
		dir, found := loader.FindFolder(".atkins", subDir)
		assert.True(t, found)
		assert.Equal(t, tmpDir, dir)
	})

	t.Run("finds closest folder when nested", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		deepDir := filepath.Join(subDir, "deep")
		require.NoError(t, os.MkdirAll(deepDir, 0o755))

		// Create .atkins at both levels
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".atkins"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(subDir, ".atkins"), 0o755))

		// Should find the closest one (in subDir)
		loader := runner.NewSkillsLoader(tmpDir, deepDir)
		dir, found := loader.FindFolder(".atkins", deepDir)
		assert.True(t, found)
		assert.Equal(t, subDir, dir)
	})

	t.Run("returns false when not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		_, found := loader.FindFolder(".atkins", tmpDir)
		assert.False(t, found)
	})
}

// TestFindFile tests the FindFile method for locating files by pattern,
// traversing parent directories.
func TestFindFile(t *testing.T) {
	t.Run("finds file in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		composePath := filepath.Join(tmpDir, "compose.yml")
		require.NoError(t, os.WriteFile(composePath, []byte("version: '3'"), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		matchDir, found := loader.FindFile([]string{"compose.yml"}, tmpDir)
		assert.True(t, found)
		assert.Equal(t, tmpDir, matchDir)
	})

	t.Run("finds file in parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "src", "app")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		composePath := filepath.Join(tmpDir, "compose.yml")
		require.NoError(t, os.WriteFile(composePath, []byte("version: '3'"), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, subDir)
		matchDir, found := loader.FindFile([]string{"compose.yml"}, subDir)
		assert.True(t, found)
		assert.Equal(t, tmpDir, matchDir)
	})

	t.Run("finds closest file when multiple exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		deepDir := filepath.Join(subDir, "deep")
		require.NoError(t, os.MkdirAll(deepDir, 0o755))

		// Create compose.yml at both levels
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "compose.yml"), []byte("root"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "compose.yml"), []byte("sub"), 0o644))

		// Should find the closest one (in subDir)
		loader := runner.NewSkillsLoader(tmpDir, deepDir)
		matchDir, found := loader.FindFile([]string{"compose.yml"}, deepDir)
		assert.True(t, found)
		assert.Equal(t, subDir, matchDir)
	})

	t.Run("matches first pattern in list", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(""), 0o644))

		// compose.yml doesn't exist, but docker-compose.yml does
		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		matchDir, found := loader.FindFile([]string{"compose.yml", "docker-compose.yml"}, tmpDir)
		assert.True(t, found)
		assert.Equal(t, tmpDir, matchDir)
	})

	t.Run("prefers closer file over pattern order", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		// docker-compose.yml in parent, compose.yml in subDir
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "compose.yml"), []byte(""), 0o644))

		// Should find compose.yml in subDir (closer), not docker-compose.yml in parent
		loader := runner.NewSkillsLoader(tmpDir, subDir)
		matchDir, found := loader.FindFile([]string{"compose.yml", "docker-compose.yml"}, subDir)
		assert.True(t, found)
		assert.Equal(t, subDir, matchDir)
	})

	t.Run("returns false when no files match", func(t *testing.T) {
		tmpDir := t.TempDir()
		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		_, found := loader.FindFile([]string{"compose.yml"}, tmpDir)
		assert.False(t, found)
	})

	t.Run("handles absolute paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		absPath := filepath.Join(tmpDir, "specific.yml")
		require.NoError(t, os.WriteFile(absPath, []byte(""), 0o644))

		// Should find absolute path directly
		loader := runner.NewSkillsLoader(tmpDir, "/some/other/dir")
		matchDir, found := loader.FindFile([]string{absPath}, "/some/other/dir")
		assert.True(t, found)
		assert.Equal(t, tmpDir, matchDir)
	})
}

// TestSkillsLoader tests the SkillsLoader for loading and evaluating skills.
func TestSkillsLoader(t *testing.T) {
	t.Run("loads skill without when condition", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Create a skill without when: condition
		skillContent := `name: Test Skill
jobs:
  default:
    cmd: echo hello
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "test.yml"), []byte(skillContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		assert.Equal(t, "test", pipelines[0].ID)
		// Dir should be the folder containing .atkins/
		assert.Equal(t, tmpDir, pipelines[0].Dir)
	})

	t.Run("loads skill with matching when condition", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "app")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Create compose.yml in app/
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "compose.yml"), []byte(""), 0o644))

		// Create a skill with when: files: condition
		skillContent := `name: Compose Skill
when:
  files:
    - compose.yml
jobs:
  up:
    cmd: docker compose up
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "compose.yml"), []byte(skillContent), 0o644))

		// Load from subDir - should find compose.yml there
		loader := runner.NewSkillsLoader(tmpDir, subDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		assert.Equal(t, "compose", pipelines[0].ID)
		// Dir should be where compose.yml was found
		assert.Equal(t, subDir, pipelines[0].Dir)
	})

	t.Run("skips skill with non-matching when condition", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Create a skill requiring compose.yml (which doesn't exist)
		skillContent := `name: Compose Skill
when:
  files:
    - compose.yml
jobs:
  up:
    cmd: docker compose up
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "compose.yml"), []byte(skillContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		assert.Len(t, pipelines, 0)
	})

	t.Run("sets Dir to match location for when files", func(t *testing.T) {
		tmpDir := t.TempDir()
		appDir := filepath.Join(tmpDir, "project", "app")
		require.NoError(t, os.MkdirAll(appDir, 0o755))

		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Create compose.yml deep in the tree
		require.NoError(t, os.WriteFile(filepath.Join(appDir, "compose.yml"), []byte(""), 0o644))

		skillContent := `name: Compose
when:
  files:
    - compose.yml
jobs:
  up:
    cmd: docker compose up
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "compose.yml"), []byte(skillContent), 0o644))

		// Start search from appDir
		loader := runner.NewSkillsLoader(tmpDir, appDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		// Should use appDir where compose.yml was found
		assert.Equal(t, appDir, pipelines[0].Dir)
	})

	t.Run("local skills override global by ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		homeDir := t.TempDir()

		localSkills := filepath.Join(tmpDir, ".atkins", "skills")
		globalSkills := filepath.Join(homeDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(localSkills, 0o755))
		require.NoError(t, os.MkdirAll(globalSkills, 0o755))

		// Same skill ID in both locations
		localContent := `name: Local Go
jobs:
  test:
    cmd: go test ./...
`
		globalContent := `name: Global Go
jobs:
  test:
    cmd: go test -v ./...
`
		require.NoError(t, os.WriteFile(filepath.Join(localSkills, "go.yml"), []byte(localContent), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(globalSkills, "go.yml"), []byte(globalContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		loader.AddSkillsDir(globalSkills) // Add global as secondary
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		assert.Equal(t, "Local Go", pipelines[0].Name) // Local takes precedence
	})

	t.Run("multiple when patterns use closest match", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// docker-compose.yml in root, compose.yml in sub
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "compose.yml"), []byte(""), 0o644))

		skillContent := `name: Compose
when:
  files:
    - compose.yml
    - docker-compose.yml
jobs:
  up:
    cmd: docker compose up
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "compose.yml"), []byte(skillContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, subDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		// Should use subDir (closest match with compose.yml)
		assert.Equal(t, subDir, pipelines[0].Dir)
	})
}

// TestSkillsLoaderWorkDirRules tests the working directory resolution rules
// as specified in docs/skills-and-workspaces.md
func TestSkillsLoaderWorkDirRules(t *testing.T) {
	t.Run("workspace skill without when uses atkins folder parent", func(t *testing.T) {
		// Scenario: /project/.atkins/skills/deploy.yml without when:
		// Working dir should be /project
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "src", "app")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		skillContent := `name: Deploy
jobs:
  default:
    cmd: ./deploy.sh
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "deploy.yml"), []byte(skillContent), 0o644))

		// Invoke from deep subdir
		loader := runner.NewSkillsLoader(tmpDir, subDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		// Dir should be tmpDir (parent of .atkins/)
		assert.Equal(t, tmpDir, pipelines[0].Dir)
	})

	t.Run("skill with when uses matched file location", func(t *testing.T) {
		// Scenario: compose skill with when: files: [compose.yml]
		// /project/app/compose.yml exists
		// Working dir should be /project/app
		tmpDir := t.TempDir()
		appDir := filepath.Join(tmpDir, "app")
		require.NoError(t, os.MkdirAll(appDir, 0o755))

		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(appDir, "compose.yml"), []byte(""), 0o644))

		skillContent := `name: Compose
when:
  files:
    - compose.yml
jobs:
  up:
    cmd: docker compose up -d
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "compose.yml"), []byte(skillContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, appDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		assert.Equal(t, appDir, pipelines[0].Dir)
	})

	t.Run("preserves existing pipeline Dir if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsDir := filepath.Join(tmpDir, ".atkins", "skills")
		require.NoError(t, os.MkdirAll(skillsDir, 0o755))

		// Skill with explicit dir: setting
		skillContent := `name: Fixed Dir Skill
dir: /custom/path
jobs:
  default:
    cmd: echo hello
`
		require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "fixed.yml"), []byte(skillContent), 0o644))

		loader := runner.NewSkillsLoader(tmpDir, tmpDir)
		pipelines, err := loader.Load()

		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		// Should preserve the explicit dir from the skill file
		assert.Equal(t, "/custom/path", pipelines[0].Dir)
	})
}
