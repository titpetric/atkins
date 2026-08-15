package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/titpetric/atkins/colors"
)

// Menu edits a configuration document interactively.
type Menu struct {
	// Path is the document being edited.
	Path string

	config *Config
	in     *bufio.Reader
	out    io.Writer

	// dirty tracks unsaved edits so quitting can warn.
	dirty bool
}

// NewMenu opens the document at path for editing, starting from the
// embedded defaults when it doesn't exist yet.
func NewMenu(path string) (*Menu, error) {
	config, err := LoadFile(path)
	if err != nil {
		return nil, err
	}

	return &Menu{
		Path:   path,
		config: config,
		in:     bufio.NewReader(os.Stdin),
		out:    os.Stdout,
	}, nil
}

// SetIO redirects the menu, so it can be driven from something other
// than a terminal.
func (m *Menu) SetIO(in io.Reader, out io.Writer) {
	m.in = bufio.NewReader(in)
	m.out = out
}

// Run drives the menu until the user quits.
func (m *Menu) Run() error {
	fields := m.config.Fields()

	for {
		m.render(fields)

		choice, err := m.prompt("Number to edit, (w)rite, (r)eset, (q)uit: ")
		if err != nil {
			return err
		}

		switch strings.ToLower(choice) {
		case "q", "quit", "":
			if m.dirty {
				confirm, err := m.prompt("Discard unsaved changes? [y/N]: ")
				if err != nil {
					return err
				}
				if !strings.EqualFold(confirm, "y") {
					continue
				}
			}
			return nil

		case "w", "write", "save":
			if err := m.write(); err != nil {
				m.fail(err)
				continue
			}
			fmt.Fprintf(m.out, "\n%s Wrote %s\n", colors.BrightGreen("✓"), m.Path)
			return nil

		case "r", "reset":
			if err := m.reset(&fields); err != nil {
				m.fail(err)
			}
			continue
		}

		index, err := strconv.Atoi(choice)
		if err != nil || index < 1 || index > len(fields) {
			m.fail(fmt.Errorf("pick a number between 1 and %d", len(fields)))
			continue
		}

		if err := m.edit(fields[index-1]); err != nil {
			m.fail(err)
		}
	}
}

// render prints the document as a numbered list.
func (m *Menu) render(fields []Field) {
	fmt.Fprintf(m.out, "\n%s\n", colors.BrightWhite("atkins configuration"))
	fmt.Fprintf(m.out, "%s\n\n", colors.Dim(m.Path))

	section := ""
	for i, field := range fields {
		if prefix, _, _ := strings.Cut(field.Path, "."); prefix != section {
			section = prefix
			fmt.Fprintf(m.out, "%s\n", colors.BrightCyan(section))
		}

		name := strings.TrimPrefix(field.Path, section+".")
		fmt.Fprintf(m.out, "  %2d) %-20s %s", i+1, name, field.Display())

		// An env override wins at runtime, so an edit here would look
		// like it did nothing. Say so up front.
		if field.Env != "" && os.Getenv(field.Env) != "" {
			fmt.Fprintf(m.out, "  %s", colors.BrightYellow("← "+field.Env))
		}
		fmt.Fprintln(m.out)
	}

	if m.dirty {
		fmt.Fprintf(m.out, "\n%s\n", colors.BrightYellow("unsaved changes"))
	}
	fmt.Fprintln(m.out)
}

// edit prompts for one field's new value.
func (m *Menu) edit(field Field) error {
	fmt.Fprintf(m.out, "\n%s\n", colors.BrightWhite(field.Path))
	fmt.Fprintf(m.out, "  current: %s\n", field.Display())
	fmt.Fprintf(m.out, "  expects: %s\n", colors.Dim(field.Kind()))
	if field.Env != "" {
		fmt.Fprintf(m.out, "  overridden by: %s\n", colors.Dim(field.Env))
	}

	value, err := m.prompt("  new value (empty clears): ")
	if err != nil {
		return err
	}

	if err := field.Set(value); err != nil {
		return err
	}

	m.dirty = true
	return nil
}

// reset returns the document to the embedded defaults.
func (m *Menu) reset(fields *[]Field) error {
	confirm, err := m.prompt("Reset every value to its default? [y/N]: ")
	if err != nil {
		return err
	}
	if !strings.EqualFold(confirm, "y") {
		return nil
	}

	config, err := Default()
	if err != nil {
		return err
	}

	m.config = config
	*fields = m.config.Fields()
	m.dirty = true

	return nil
}

// write validates and saves the document.
func (m *Menu) write() error {
	if err := m.config.Validate(); err != nil {
		return err
	}
	if err := m.config.Save(m.Path); err != nil {
		return err
	}

	m.dirty = false
	return nil
}

// prompt reads one line.
func (m *Menu) prompt(label string) (string, error) {
	fmt.Fprint(m.out, label)

	line, err := m.in.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			// A piped stdin that ran out means "done"; treat it as
			// quit rather than erroring on every subsequent read.
			return "q", nil
		}
		return "", err
	}

	return strings.TrimSpace(line), nil
}

// fail reports a problem without leaving the menu.
func (m *Menu) fail(err error) {
	fmt.Fprintf(m.out, "\n%s %v\n", colors.BrightRed("ERROR:"), err)
}
