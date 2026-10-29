// Package markdown finds structure in a rendered markdown document: the
// blank-line-separated blocks of the document, and the GFM pipe
// tables among them.
//
// It serves one caller: `atkins --help` builds one long markdown
// document, and draws each table in it as a bordered, coloured grid in
// the terminal. Sections splits the document into blocks, and Render
// replaces each table block in place and keeps the prose around it. A fenced
// code block (a line starting with ``` opens and closes it) stays one
// section when it holds a blank line. A YAML example stays in one piece.
//
// Render redraws every pipe table as a bordered ANSI table, colours every
// ATX heading the caller left plain, and leaves the rest of the document
// unchanged. With colour off it returns the document byte for byte, and
// `atkins --help > help.md` or a piped `atkins --help | less` gets plain
// markdown a table-aware reader can parse.
package markdown
