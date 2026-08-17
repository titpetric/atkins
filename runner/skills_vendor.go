package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/titpetric/atkins/model"
)

// VendorStatus is what installing a skill would do to the repository.
type VendorStatus string

const (
	// VendorNew means the repository doesn't carry the skill yet.
	VendorNew VendorStatus = "new"

	// VendorCurrent means an identical copy is already vendored.
	VendorCurrent VendorStatus = "up to date"

	// VendorChanged means the vendored copy differs from the source and
	// would be overwritten.
	VendorChanged VendorStatus = "changed"
)

// VendorSkill is a candidate skill file, with the reason it was picked
// for vendoring once it has been matched against a repository.
type VendorSkill struct {
	// ID is the skill namespace, taken from the file name.
	ID string

	// Path is the absolute path of the source skill file.
	Path string

	// Pipeline is the parsed skill.
	Pipeline *model.Pipeline

	// Reason records why the skill is used here, e.g. the path that
	// satisfied its when: block, or the pipeline that references it.
	Reason string

	// Status is what installing this skill would do, set for used skills.
	Status VendorStatus

	// Added and Removed are the line counts installing this skill would
	// add to and remove from the vendored copy.
	Added   int
	Removed int

	// Diff is the unified diff from the vendored copy to the source,
	// empty when there is nothing to change.
	Diff string
}

// VendorResult reports what a vendoring run found, matched and wrote.
type VendorResult struct {
	// SourceDir is the skills directory that was read.
	SourceDir string

	// TargetDir is the repository directory that was written to.
	TargetDir string

	// Found are all skills available in SourceDir.
	Found []*VendorSkill

	// Used are the skills this repository has a use for.
	Used []*VendorSkill

	// Installed are the IDs written to TargetDir. It stays empty on a
	// dry run; Pending says what a write would have done.
	Installed []string
}

// Pending returns the IDs a write would create or overwrite, leaving
// out the skills already vendored unchanged.
func (r *VendorResult) Pending() []string {
	var pending []string
	for _, skill := range r.Used {
		if skill.Status != VendorCurrent {
			pending = append(pending, skill.ID)
		}
	}
	return pending
}

// vendorSkipDirs are directory names not descended into when looking for
// when: matches and pipeline files. Dot directories are skipped too, so
// the repository's own .atkins/ never matches a skill against itself.
var vendorSkipDirs = []string{
	"node_modules",
	"vendor",
	"testdata",
	"bin",
	"dist",
	"coverage",
}

// Vendorer copies the skills a repository uses out of a shared skills
// directory and into the repository itself, so a clone or a CI agent
// gets the same jobs without a personal $HOME/.atkins/skills.
type Vendorer struct {
	// SourceDir is the skills directory to vendor from, typically
	// $HOME/.atkins/skills.
	SourceDir string

	// Root is the repository root, the folder holding .git.
	Root string

	// TargetDir is where skills are written, Root/.atkins/skills.
	TargetDir string

	// SkipDirs are directory names not descended into.
	SkipDirs []string
}

// NewVendorer creates a vendorer writing into root/.atkins/skills.
func NewVendorer(sourceDir, root string) *Vendorer {
	return &Vendorer{
		SourceDir: sourceDir,
		Root:      root,
		TargetDir: filepath.Join(root, ".atkins", "skills"),
		SkipDirs:  vendorSkipDirs,
	}
}

// FindRepositoryRoot returns the folder holding .git, starting at
// startDir and traversing parent directories.
func FindRepositoryRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		// A worktree or a submodule has .git as a file, not a folder.
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found above %s", startDir)
		}
		dir = parent
	}
}

// Run plans the vendoring and installs what it selected.
//
// Plan on its own is the dry run: it reports the same selection without
// touching the repository.
func (v *Vendorer) Run() (*VendorResult, error) {
	result, err := v.Plan()
	if err != nil {
		return nil, err
	}
	if err := v.Install(result); err != nil {
		return result, err
	}
	return result, nil
}

// Plan reads the source skills and decides which of them this
// repository uses, without writing anything.
//
// A skill is used when its when: block matches a path anywhere in the
// repository, when it has no when: block at all (it is active
// everywhere), or when a pipeline in the repository - or another
// selected skill - references one of its jobs.
func (v *Vendorer) Plan() (*VendorResult, error) {
	if v.SourceDir == v.TargetDir {
		return nil, fmt.Errorf("source and target are the same directory: %s", v.SourceDir)
	}

	result := &VendorResult{
		SourceDir: v.SourceDir,
		TargetDir: v.TargetDir,
	}

	candidates, err := v.candidates()
	if err != nil {
		return nil, err
	}
	result.Found = candidates

	known := make(map[string]*VendorSkill, len(candidates))
	for _, candidate := range candidates {
		known[candidate.ID] = candidate
	}

	dirs, err := v.walk()
	if err != nil {
		return nil, err
	}

	used := make(map[string]string)
	var queue []string

	use := func(id, reason string) {
		if _, ok := known[id]; !ok {
			return
		}
		if _, ok := used[id]; ok {
			return
		}
		used[id] = reason
		queue = append(queue, id)
	}

	// Skills the workspace activates on its own.
	for _, candidate := range candidates {
		if reason, ok := v.matchWhen(candidate.Pipeline, dirs); ok {
			use(candidate.ID, reason)
		}
	}

	// Skills the repository's pipelines name directly.
	for _, path := range v.pipelineFiles(dirs) {
		pipelines, err := LoadPipeline(path)
		if err != nil {
			// A pipeline that doesn't parse is a problem for the run
			// that uses it, not for vendoring; take what we can read.
			continue
		}

		label := path
		if rel, err := filepath.Rel(v.Root, path); err == nil {
			label = rel
		}

		for _, pipeline := range pipelines {
			for _, id := range skillRefs(pipeline) {
				use(id, "referenced by "+label)
			}
		}
	}

	// Skills the selected skills themselves reach for, e.g. a release
	// skill calling docker:build.
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		for _, ref := range skillRefs(known[id].Pipeline) {
			use(ref, "referenced by "+id)
		}
	}

	for _, candidate := range candidates {
		reason, ok := used[candidate.ID]
		if !ok {
			continue
		}

		candidate.Reason = reason
		v.compare(candidate)
		result.Used = append(result.Used, candidate)
	}

	return result, nil
}

// compare diffs a skill against the copy the repository carries, and
// records what installing it would change.
func (v *Vendorer) compare(skill *VendorSkill) {
	target := filepath.Join(v.TargetDir, filepath.Base(skill.Path))

	existing, readErr := os.ReadFile(target)

	data, err := os.ReadFile(skill.Path)
	if err != nil {
		// Install reports the read error properly; treat it as work to do.
		skill.Status = VendorChanged
		return
	}

	from, to := diffLines(existing), diffLines(data)

	switch {
	case readErr != nil:
		// Nothing to diff against, so the whole file is the addition.
		skill.Status = VendorNew
		skill.Added = len(to)
		return
	case bytes.Equal(existing, data):
		skill.Status = VendorCurrent
		return
	default:
		skill.Status = VendorChanged
	}

	for _, op := range difflib.NewMatcher(from, to).GetOpCodes() {
		switch op.Tag {
		case 'r':
			skill.Removed += op.I2 - op.I1
			skill.Added += op.J2 - op.J1
		case 'd':
			skill.Removed += op.I2 - op.I1
		case 'i':
			skill.Added += op.J2 - op.J1
		}
	}

	fromLabel := target
	if rel, err := filepath.Rel(v.Root, target); err == nil {
		fromLabel = rel
	}

	// A diff that can't be rendered is not worth failing over; the line
	// counts above are what the caller reports either way.
	skill.Diff, _ = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        from,
		B:        to,
		FromFile: fromLabel,
		ToFile:   skill.Path,
		Context:  3,
	})
}

// diffLines splits a file into lines that each keep their newline, with
// no phantom line for the trailing one.
func diffLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	lines := strings.SplitAfter(string(data), "\n")
	if last := len(lines) - 1; lines[last] == "" {
		return lines[:last]
	}

	return lines
}

// Install copies the selected skills into TargetDir, creating it when it
// doesn't exist, and overwrites a vendored copy that has drifted from
// its source. Vendored skills are tracked files like any other, so
// reverting an unwanted change is the user's call, not this command's.
func (v *Vendorer) Install(result *VendorResult) error {
	if len(result.Pending()) == 0 {
		return nil
	}

	if err := os.MkdirAll(v.TargetDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", v.TargetDir, err)
	}

	for _, skill := range result.Used {
		if skill.Status == VendorCurrent {
			continue
		}

		data, err := os.ReadFile(skill.Path)
		if err != nil {
			return fmt.Errorf("failed to read skill %s: %w", skill.Path, err)
		}

		target := filepath.Join(v.TargetDir, filepath.Base(skill.Path))
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", target, err)
		}

		result.Installed = append(result.Installed, skill.ID)
	}

	return nil
}

// candidates loads every skill file in the source directory, sorted by ID.
func (v *Vendorer) candidates() ([]*VendorSkill, error) {
	entries, err := os.ReadDir(v.SourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read skills directory %s: %w", v.SourceDir, err)
	}

	loader := &SkillsLoader{}
	var skills []*VendorSkill

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(v.SourceDir, entry.Name())
		pipeline, err := loader.loadSkillFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load skill %s: %w", path, err)
		}

		skills = append(skills, &VendorSkill{
			ID:       pipeline.ID,
			Path:     path,
			Pipeline: pipeline,
		})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].ID < skills[j].ID
	})

	return skills, nil
}

// walk returns every directory in the repository worth matching against,
// the root first, skipping dot directories and SkipDirs.
func (v *Vendorer) walk() ([]string, error) {
	skip := make(map[string]bool, len(v.SkipDirs))
	for _, name := range v.SkipDirs {
		skip[name] = true
	}

	var dirs []string
	err := filepath.WalkDir(v.Root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			// An unreadable directory is not worth failing the run over.
			return nil //nolint:nilerr
		}
		if !entry.IsDir() {
			return nil
		}
		if path == v.Root {
			dirs = append(dirs, path)
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") || skip[entry.Name()] {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", v.Root, err)
	}

	return dirs, nil
}

// matchWhen reports whether a skill's when: block is satisfied anywhere
// in the repository, and by what. A skill without when: is active
// everywhere, so it matches by definition.
func (v *Vendorer) matchWhen(pipeline *model.Pipeline, dirs []string) (reason string, matched bool) {
	if pipeline.When == nil || len(pipeline.When.Files) == 0 {
		return "always active", true
	}

	for _, pattern := range pipeline.When.Files {
		if !filepath.IsAbs(pattern) {
			continue
		}
		if match, ok := MatchWhenPattern("", pattern); ok {
			return match, true
		}
	}

	for _, dir := range dirs {
		for _, pattern := range pipeline.When.Files {
			if filepath.IsAbs(pattern) {
				continue
			}

			match, ok := MatchWhenPattern(dir, pattern)
			if !ok {
				continue
			}
			if rel, err := filepath.Rel(v.Root, match); err == nil {
				return rel, true
			}
			return match, true
		}
	}

	return "", false
}

// pipelineFiles returns the pipeline files carried by the repository:
// the config files in the walked directories, plus the project's own
// skills, which the walk skips along with the rest of .atkins/.
func (v *Vendorer) pipelineFiles(dirs []string) []string {
	var files []string

	for _, dir := range dirs {
		for _, name := range ConfigNames {
			candidate := filepath.Join(dir, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				files = append(files, candidate)
			}
		}
	}

	entries, err := os.ReadDir(v.TargetDir)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		files = append(files, filepath.Join(v.TargetDir, entry.Name()))
	}

	return files
}

// skillRefs returns the skill IDs a pipeline names through task: and
// depends_on:, in the order they are first seen.
func skillRefs(pipeline *model.Pipeline) []string {
	jobs := pipeline.GetJobs()
	seen := make(map[string]bool)

	var refs []string
	add := func(ref string) {
		id, ok := skillRefID(ref, jobs)
		if !ok || seen[id] {
			return
		}
		seen[id] = true
		refs = append(refs, id)
	}

	for _, job := range jobs {
		for _, dep := range job.DependsOn {
			add(dep)
		}
		for _, step := range job.Children() {
			if step.Task != "" {
				add(step.Task)
			}
		}
	}

	sort.Strings(refs)
	return refs
}

// skillRefID extracts the skill namespace from a task or dependency
// reference. A bare name is only a skill reference when the pipeline
// has no job of its own by that name.
func skillRefID(ref string, jobs map[string]*model.Job) (string, bool) {
	ref = strings.TrimPrefix(strings.TrimSpace(ref), ":")
	if ref == "" {
		return "", false
	}

	if index := strings.Index(ref, ":"); index > 0 {
		return ref[:index], true
	}

	if _, local := jobs[ref]; local {
		return "", false
	}

	return ref, true
}
