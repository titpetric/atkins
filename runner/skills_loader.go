package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// SkillsLoader discovers and loads skill pipelines from .atkins/skills/ directories.
// It checks `when:` conditions to determine which skills apply and sets
// the appropriate working directory for each skill based on the rules.
type SkillsLoader struct {
	// SkillsDirs are the directories to search for skill files, in priority order.
	// First directory takes precedence for skills with the same ID.
	SkillsDirs []string

	// StartDir is the directory from which to start searching for when: files.
	// This is typically the user's working directory.
	StartDir string

	// WorkspaceDir is the folder containing .atkins/ (used for skills without when:).
	WorkspaceDir string

	// Owners maps a skills directory to the folder holding its .atkins/,
	// where a skill without when: from that directory runs. A directory
	// missing from the map runs its skills in WorkspaceDir.
	Owners map[string]string
}

// NewSkillsLoader creates a loader for the given workspace.
// workspaceDir is the folder containing .atkins/ (used as Dir for skills without when:).
// startDir is where to start searching for when: files (typically user's cwd).
func NewSkillsLoader(workspaceDir, startDir string) *SkillsLoader {
	return &SkillsLoader{
		SkillsDirs:   []string{filepath.Join(workspaceDir, ".atkins", "skills")},
		StartDir:     startDir,
		WorkspaceDir: workspaceDir,
	}
}

// AddSkillsDir adds another skills directory to search.
// Directories added later have lower precedence.
func (l *SkillsLoader) AddSkillsDir(dir string) {
	l.SkillsDirs = append(l.SkillsDirs, dir)
}

// AddWorkspace adds dir/.atkins/skills to search, at lower precedence,
// and runs its skills without when: in dir.
func (l *SkillsLoader) AddWorkspace(dir string) {
	skillsDir := filepath.Join(dir, ".atkins", "skills")
	if l.Owners == nil {
		l.Owners = make(map[string]string)
	}
	l.Owners[skillsDir] = dir
	l.AddSkillsDir(skillsDir)
}

// AddParentWorkspaces adds the .atkins/skills of every folder above
// WorkspaceDir, nearest first, so a project cloned inside a workspace
// checkout sees the workspace's skills. See ParentWorkspaces for where
// the walk stops.
func (l *SkillsLoader) AddParentWorkspaces() {
	home, _ := os.UserHomeDir()
	for _, dir := range ParentWorkspaces(l.WorkspaceDir, home) {
		l.AddWorkspace(dir)
	}
}

// ParentWorkspaces returns the folders above dir holding .atkins/skills,
// nearest first.
//
// Inside home the walk stops below home: $HOME/.atkins/skills loads on
// its own as the global skills, and the folders above home are not the
// user's to configure. Outside home the walk goes on to the filesystem
// root, the same as DiscoverConfig.
func ParentWorkspaces(dir, home string) []string {
	dir = filepath.Clean(dir)
	if home != "" {
		home = filepath.Clean(home)
	}

	var dirs []string
	for current := dir; ; {
		parent := filepath.Dir(current)
		if parent == current || current == home {
			return dirs
		}
		current = parent

		if current == home {
			continue
		}
		if info, err := os.Stat(filepath.Join(current, ".atkins", "skills")); err == nil && info.IsDir() {
			dirs = append(dirs, current)
		}
	}
}

// Load discovers and returns all enabled skill pipelines.
//
// A skill from an earlier directory shadows one of the same ID found
// later, so a project skill wins over the global one it shares a name
// with. A skill whose when: block does not match contributes nothing and
// does not shadow anything.
func (l *SkillsLoader) Load() ([]*model.Pipeline, error) {
	skills, err := l.Scan()
	if err != nil {
		return nil, err
	}
	return activePipelines(skills), nil
}

// activePipelines returns the pipelines of the active skills, the first
// one for each ID.
func activePipelines(skills []*Skill) []*model.Pipeline {
	seen := make(map[string]bool)

	var pipelines []*model.Pipeline
	for _, skill := range skills {
		if !skill.Active || seen[skill.ID()] {
			continue
		}
		seen[skill.ID()] = true
		pipelines = append(pipelines, skill.Pipeline)
	}

	return pipelines
}

// LoadAll returns the skill pipelines Load returns, followed by the
// skills whose when: block does not match, marked Inactive.
//
// An inactive skill shadows nothing: it appears only for an ID no
// active skill carries, once, from the highest-priority directory.
// `atkins -l --json --all` lists them so a caller sees every skill
// installed for a folder; a run never uses LoadAll.
func (l *SkillsLoader) LoadAll() ([]*model.Pipeline, error) {
	skills, err := l.Scan()
	if err != nil {
		return nil, err
	}
	return MergeSkills(activePipelines(skills), inactivePipelines(skills)), nil
}

// inactivePipelines returns the pipelines of the skills that are not
// active, marked Inactive.
func inactivePipelines(skills []*Skill) []*model.Pipeline {
	var pipelines []*model.Pipeline
	for _, skill := range skills {
		if skill.Active {
			continue
		}
		skill.Pipeline.Inactive = true
		pipelines = append(pipelines, skill.Pipeline)
	}
	return pipelines
}

// MergeSkills appends the skill pipelines in more to pipelines, skipping
// an ID pipelines already carries. An active pipeline from more takes
// the place of an inactive one with the same ID, so a skill that applies
// wins over one that does not, whatever directory each comes from.
func MergeSkills(pipelines, more []*model.Pipeline) []*model.Pipeline {
	index := make(map[string]int)
	for i, p := range pipelines {
		if p.ID != "" {
			if _, ok := index[p.ID]; !ok {
				index[p.ID] = i
			}
		}
	}

	for _, p := range more {
		i, ok := index[p.ID]
		switch {
		case !ok:
			index[p.ID] = len(pipelines)
			pipelines = append(pipelines, p)
		case pipelines[i].Inactive && !p.Inactive:
			pipelines[i] = p
		}
	}
	return pipelines
}

// Scan returns every skill file in the loader's directories, in
// directory order, whether its when: block matches or not.
//
// Load drops the skills that do not apply here; Scan keeps them so
// `atkins --help` can list every installed skill, beyond the ones the
// current directory activates. Two directories carrying the same skill
// ID both appear, the higher-priority one first.
func (l *SkillsLoader) Scan() ([]*Skill, error) {
	var skills []*Skill

	for _, skillsDir := range l.SkillsDirs {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read skills directory %s: %w", skillsDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
				continue
			}

			skillPath := filepath.Join(skillsDir, entry.Name())
			skill, err := l.loadSkill(skillPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load skill %s: %w", skillPath, err)
			}

			skill.Dir, skill.Matched, skill.Active = l.evaluateWhen(skill.Pipeline, l.owner(skillsDir))
			skill.Pipeline.Matched = skill.Matched

			// Set Dir only if not already explicitly set in the skill file
			if skill.Active && skill.Pipeline.Dir == "" {
				skill.Pipeline.Dir = skill.Dir
			}

			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// loadSkill reads a skill file and the optional markdown companion
// beside it: go.md in the same directory documents go.yml.
// A missing or unreadable companion leaves Doc empty and is not an
// error, because the guide is optional.
func (l *SkillsLoader) loadSkill(path string) (*Skill, error) {
	pipeline, err := l.loadSkillFile(path)
	if err != nil {
		return nil, err
	}

	skill := &Skill{Pipeline: pipeline, Path: path}

	docPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".md"
	if data, err := os.ReadFile(docPath); err == nil {
		skill.DocPath = docPath
		skill.Doc = strings.TrimSpace(string(data))
	}

	return skill, nil
}

// loadSkillFile loads a single skill pipeline from a YAML file.
// Sets Pipeline.ID from the filename (e.g., "go.yml" -> "go").
func (l *SkillsLoader) loadSkillFile(path string) (*model.Pipeline, error) {
	pipelines, err := LoadPipeline(path)
	if err != nil {
		return nil, err
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("no pipeline found in skill file %s", path)
	}

	pipeline := pipelines[0]
	filename := filepath.Base(path)
	pipeline.ID = strings.TrimSuffix(filename, filepath.Ext(filename))

	return pipeline, nil
}

// owner returns the folder a skill without when: from skillsDir runs in.
func (l *SkillsLoader) owner(skillsDir string) string {
	if dir, ok := l.Owners[skillsDir]; ok {
		return dir
	}
	return l.WorkspaceDir
}
