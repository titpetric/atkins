package ai

import (
	"os"
	"strings"

	"github.com/titpetric/atkins/model"
	"github.com/titpetric/atkins/runner"
)

// instructions is the static half of the prompt. It contains a literal %s
// and braces, so it is written to the prompt as-is and never through
// fmt.Sprintf.
const instructions = `Desired response: One or more commands to run in json format, ` +
	`{"cmds": ["atkins", "atkins release"]} or whatever the user asked for. ` +
	`If the user explicitly asks to create new files or atkins.yml, then do as instructed. ` +
	`Reply with {"message": "what you did"}, the message can contain whitespace but keep it short, ` +
	`e.g. "Added new %s target to atkins.yml, added/updated/moved/ {filename}". ` +
	`Don't list all filenames if you can just state the change folder, e.g. ` +
	`"verb(docs): git commit short title only". ` +
	`If no change is requested, if no job can reasonably be inferred from the request, ` +
	`fall back to {"cmds": ["atkins -l"]} to list available jobs.`

// BuildPrompt builds the prompt sent to `claude -p`. It carries the input,
// the job listing `atkins -l` prints, and the atkins.yml discovered from
// workDir.
func BuildPrompt(userInput string, pipelines []*model.Pipeline, workDir string) string {
	var b strings.Builder
	b.WriteString("The user writes: ")
	b.WriteString(userInput)
	b.WriteString("\n\nThe pipelines available are:\n\n")
	b.WriteString(runner.ListPipelines(pipelines))
	b.WriteString("\nAtkins pipeline: ")
	b.WriteString(readConfig(workDir))
	b.WriteString("\n\n")
	b.WriteString(instructions)
	return b.String()
}

// readConfig returns the atkins.yml found from workDir, or a placeholder
// line when there is none to read.
func readConfig(workDir string) string {
	configPath, _, err := runner.DiscoverConfig(workDir)
	if err != nil || configPath == "" {
		return "(no atkins.yml found)"
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return "(no atkins.yml found)"
	}

	return string(content)
}
