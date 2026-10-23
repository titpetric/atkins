// Package markdown finds structure in a rendered markdown document without
// a full parser: the blank-line-separated blocks a document is already
// written in, and the GFM pipe tables among them.
//
// It serves one caller: `atkins --help` builds one long markdown
// document, and a table in it reads better in a terminal as a bordered,
// coloured grid than as pipe-delimited text. Sections splits the document
// so Render can find and replace a table block without disturbing the
// prose around it. A fenced code block (a line starting with ``` opens
// and closes it) stays one section even when it holds a blank line, so a
// YAML example never splits mid-fence.
//
// Render redraws every pipe table as a bordered ANSI table, colours every
// ATX heading the caller left plain, and leaves everything else alone.
// With colour off it returns the document byte for byte, so
// `atkins --help > help.md` or a piped `atkins --help | less` still gets
// plain markdown a table-aware reader can parse.
package markdown
