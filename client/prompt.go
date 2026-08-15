package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNoTerminal is returned when a password is needed but stdin isn't a
// terminal and no environment override was provided.
var ErrNoTerminal = errors.New("stdin is not a terminal: set ATKINS_PASSWORD to log in non-interactively")

// Environment overrides for non-interactive login and registration.
// They exist so a container or a provisioning script can attach a
// machine without a human at the keyboard.
const (
	EnvEmail    = "ATKINS_EMAIL"
	EnvUsername = "ATKINS_USERNAME"
	EnvPassword = "ATKINS_PASSWORD"
)

// Prompter reads credentials from the terminal.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer

	// fd is the file descriptor used for the no-echo password read.
	fd int
}

// NewPrompter returns a Prompter over stdin and stderr.
//
// Prompts go to stderr so that `atkins --login` can be used in a
// pipeline without the prompt text landing in captured stdout.
func NewPrompter() *Prompter {
	return &Prompter{
		in:  bufio.NewReader(os.Stdin),
		out: os.Stderr,
		fd:  int(os.Stdin.Fd()),
	}
}

// NewPrompterFrom returns a Prompter reading from in and writing to out.
//
// The reader is not a terminal, so Password refuses unless the value
// comes from the environment — which is the behaviour a script gets,
// and the behaviour worth testing.
func NewPrompterFrom(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{
		in:  bufio.NewReader(in),
		out: out,
		fd:  -1,
	}
}

// Line prompts for a value, returning the trimmed input. When env names
// a set environment variable, its value is used and no prompt is shown.
func (p *Prompter) Line(label, env string) (string, error) {
	if env != "" {
		if value := os.Getenv(env); value != "" {
			return strings.TrimSpace(value), nil
		}
	}

	fmt.Fprintf(p.out, "%s: ", label)

	value, err := p.in.ReadString('\n')
	if err != nil && (value == "" || !errors.Is(err, io.EOF)) {
		return "", err
	}

	return strings.TrimSpace(value), nil
}

// Password prompts for a secret without echoing it.
func (p *Prompter) Password(label string) (string, error) {
	if value := os.Getenv(EnvPassword); value != "" {
		return value, nil
	}

	if !term.IsTerminal(p.fd) {
		return "", ErrNoTerminal
	}

	fmt.Fprintf(p.out, "%s: ", label)
	value, err := term.ReadPassword(p.fd)
	fmt.Fprintln(p.out)
	if err != nil {
		return "", err
	}

	return string(value), nil
}
