package agent

import (
	"os"
	"path/filepath"

	"github.com/titpetric/atkins/agent/router"
	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// refreshCwd updates the current working directory and git branch.
func (m *Model) refreshCwd() {
	if cwd, err := os.Getwd(); err == nil {
		m.cwd = cwd
	}
	m.gitBranch = detectGitBranch(m.cwd)
}

// changeDir handles changing the working directory and reloading pipelines.
func (m *Model) changeDir(dir string) error {
	target := dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(m.cwd, target)
	}
	target = filepath.Clean(target)

	if err := os.Chdir(target); err != nil {
		return err
	}

	m.cwd = target
	m.gitBranch = detectGitBranch(target)
	m.agent.workDir = target

	// Reload pipelines for new directory
	loader := runner.NewSkillsLoader(target, target)
	pipelines, err := loader.Load()
	if err != nil {
		pipelines = []*model.Pipeline{}
	}

	if !m.agent.options.Jail {
		if home, err := os.UserHomeDir(); err == nil {
			globalLoader := runner.NewSkillsLoader(target, target)
			globalLoader.SkillsDirs = []string{filepath.Join(home, ".atkins", "skills")}
			if globalPipelines, globalErr := globalLoader.Load(); globalErr == nil {
				seen := make(map[string]bool)
				for _, p := range pipelines {
					if p.ID != "" {
						seen[p.ID] = true
					}
				}
				for _, gp := range globalPipelines {
					if !seen[gp.ID] {
						pipelines = append(pipelines, gp)
					}
				}
			}
		}
	}

	if configPath, _, err := runner.DiscoverConfigFromCwd(); err == nil && configPath != "" {
		if mainPipelines, loadErr := runner.LoadPipeline(configPath); loadErr == nil {
			pipelines = append(mainPipelines, pipelines...)
		}
	}

	m.agent.pipelines = pipelines
	m.agent.resolver = runner.NewTaskResolver(pipelines)
	m.router = router.NewRouter(m.agent.Resolver(), m.agent.Pipelines(), m.registry)

	return nil
}
