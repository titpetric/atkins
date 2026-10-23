package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titpetric/atkins/model"
)

// SkillsLoader loads the skill pipelines of .atkins/skills/ directories,
// in priority order: a skill shadows one of the same ID from a later
// directory, so a project skill wins over a global one. A skill runs in the
// folder of the file its when: block matched, or without one, in the folder
// holding its .atkins/. A skill whose when: block does not match is inactive
// and shadows nothing.
//
// Load returns the active skills, the ones a run uses. Scan returns every
// skill for `atkins --help`, and LoadAll adds the inactive skills of an ID
// no active skill carries, for `atkins -l --json --all`.
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

// NewSkillsLoader returns a loader for workspaceDir/.atkins/skills, matching when: blocks from startDir.
func NewSkillsLoader(workspaceDir, startDir string) *SkillsLoader {
	return &SkillsLoader{
		SkillsDirs:   []string{filepath.Join(workspaceDir, ".atkins", "skills")},
		StartDir:     startDir,
		WorkspaceDir: workspaceDir,
	}
}

// AddSkillsDir adds a skills directory below the ones already there.
func (l *SkillsLoader) AddSkillsDir(dir string) {
	l.SkillsDirs = append(l.SkillsDirs, dir)
}

// AddWorkspace adds dir/.atkins/skills, whose skills without when: run in dir.
func (l *SkillsLoader) AddWorkspace(dir string) {
	skillsDir := filepath.Join(dir, ".atkins", "skills")
	if l.Owners == nil {
		l.Owners = make(map[string]string)
	}
	l.Owners[skillsDir] = dir
	l.AddSkillsDir(skillsDir)
}

// AddParentWorkspaces adds the workspaces ParentWorkspaces finds above WorkspaceDir.
func (l *SkillsLoader) AddParentWorkspaces() {
	home, _ := os.UserHomeDir()
	for _, dir := range ParentWorkspaces(l.WorkspaceDir, home) {
		l.AddWorkspace(dir)
	}
}

// ParentWorkspaces returns the folders above dir holding .atkins/skills, nearest first,
// stopping below home when dir is inside it.
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

// Load returns the error of a skill that does not load,
// or the pipelines of the active skills, one per ID.
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

// LoadAll returns the error of a skill that does not load,
// or the pipelines Load returns followed by the inactive skills, marked Inactive.
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

// MergeSkills returns pipelines with the new IDs of more appended, an active one replacing an inactive one.
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

// Scan returns every skill in directory order, active or not, a shadowed one included,
// or the error of a directory that does not read
// or a skill that does not load.
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
