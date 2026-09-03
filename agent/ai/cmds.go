package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// errUnterminatedQuote is returned by splitArgs for a command with an
// unclosed ' or ".
var errUnterminatedQuote = errors.New("unterminated quote")

// ValidateAtkinsCmds splits each suggested command into argv and rejects the
// whole batch unless every one of them invokes atkins. This is the boundary
// for AI-suggested commands: they are run as argv against the atkins binary
// with no shell involved, so a command that names anything else can only be
// refused here, never rewritten into something safe.
func ValidateAtkinsCmds(cmds []string) ([][]string, error) {
	argvs := make([][]string, 0, len(cmds))
	for _, cmd := range cmds {
		argv, err := splitArgs(cmd)
		if err != nil {
			return nil, fmt.Errorf("could not parse command suggested by AI: %q: %w", cmd, err)
		}
		// argv[0] is discarded at execution time in favour of the running
		// binary, so a path spelling like ./atkins is accepted; anything
		// with another name is not.
		if len(argv) == 0 || filepath.Base(argv[0]) != "atkins" {
			return nil, fmt.Errorf("refusing non-atkins command suggested by AI: %q", cmd)
		}
		argvs = append(argvs, argv)
	}
	return argvs, nil
}

// splitArgs splits a command line into argv on whitespace, keeping quoted
// runs together so `atkins -x "run the tests"` stays three arguments. A
// backslash escapes the next character outside single quotes.
func splitArgs(cmd string) ([]string, error) {
	var (
		argv    []string
		current strings.Builder
		quote   byte
		escaped bool
		started bool
	)

	flush := func() {
		if started {
			argv = append(argv, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		switch {
		case escaped:
			current.WriteByte(c)
			escaped = false
		case c == '\\' && quote != '\'':
			escaped = true
			started = true
		case quote != 0:
			if c == quote {
				quote = 0
				continue
			}
			current.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			started = true
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		default:
			current.WriteByte(c)
			started = true
		}
	}

	if quote != 0 || escaped {
		return nil, errUnterminatedQuote
	}
	flush()

	return argv, nil
}

// Format renders an argv back into a command line, quoting arguments that
// contain whitespace. A confirmation prompt built by joining on spaces
// would show `atkins -x run the tests` for a three-argument command.
func Format(argv []string) string {
	parts := make([]string, len(argv))
	for i, arg := range argv {
		if strings.ContainsAny(arg, " \t\n") {
			parts[i] = `"` + arg + `"`
			continue
		}
		parts[i] = arg
	}
	return strings.Join(parts, " ")
}

// Run executes validated commands against the running atkins binary, in
// order, stopping at the first failure. argv[0] is dropped: the binary is
// the one already running, so a suggestion cannot pick which program is
// started, only which arguments it gets.
func Run(ctx context.Context, workDir string, argvs [][]string, stdout, stderr io.Writer) error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	return run(ctx, binary, workDir, argvs, stdout, stderr)
}

func run(ctx context.Context, binary, workDir string, argvs [][]string, stdout, stderr io.Writer) error {
	for _, argv := range argvs {
		cmd := exec.CommandContext(ctx, binary, argv[1:]...)
		cmd.Dir = workDir
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
