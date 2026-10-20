package helpdoc

import (
	"github.com/titpetric/cli"
)

// Command is one subcommand in the help document: the word a user types,
// what it does, and the flags it defines.
type Command struct {
	// Name is the word on the command line, e.g. "server".
	Name string

	// Title is one line on what the command does.
	Title string

	// Default marks the command that runs when the user names none. Write
	// prints its flags as the global ones.
	Default bool

	// Flags are the command's own flags, nil when it has none.
	Flags *cli.FlagSet
}
