package colors

import (
	"fmt"
	"os"
	"regexp"
	"sync"

	"golang.org/x/term"
)

// stdoutIsTerminal caches the stdout terminal check for the process.
var stdoutIsTerminal = sync.OnceValue(func() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
})

// StdoutIsTerminal reports whether the process stdout is attached to a
// terminal. The check runs once; every later call returns the cached answer.
func StdoutIsTerminal() bool {
	return stdoutIsTerminal()
}

// csiPattern matches ANSI CSI escape sequences.
var csiPattern = regexp.MustCompile(`\x1b\[[?>]?[0-9;]*[A-Za-z]`)

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(in string) string {
	return csiPattern.ReplaceAllString(in, "")
}

// VisualLength returns the visual length of a string (excluding ANSI sequences).
func VisualLength(s string) int {
	stripped := StripANSI(s)
	return len([]rune(stripped))
}

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorBright  = "\033[1m"
	colorDim     = "\033[2m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorOrange  = "\033[38;5;208m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorRed     = "\033[31m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"
	// colorMediumGray is 256-color palette entry 244 (#808080), ~50% gray.
	colorMediumGray = "\033[38;5;244m"
)

func colorize(color, text string) string {
	return color + text + colorReset
}

// BrightGreen returns text in bright green color.
func BrightGreen(text string) string {
	return colorize(colorBright+colorGreen, text)
}

// Green returns text in green color.
func Green(text string) string {
	return colorize(colorGreen, text)
}

// BrightYellow returns text in bright yellow color.
func BrightYellow(text string) string {
	return colorize(colorBright+colorYellow, text)
}

// BrightOrange returns text in bright orange color.
func BrightOrange(text string) string {
	return colorize(colorOrange, text)
}

// BrightCyan returns text in bright cyan color.
func BrightCyan(text string) string {
	return colorize(colorBright+colorCyan, text)
}

// Cyan returns text in cyan color.
func Cyan(text string) string {
	return colorize(colorCyan, text)
}

// BrightMagenta returns text in bright magenta color.
func BrightMagenta(text string) string {
	return colorize(colorBright+colorMagenta, text)
}

// BrightRed returns text in bright red color.
func BrightRed(text string) string {
	return colorize(colorBright+colorRed, text)
}

// Dim returns text in dim color.
func Dim(text string) string {
	return colorize(colorDim, text)
}

// BrightWhite returns text in bright white color.
func BrightWhite(text string) string {
	return colorize(colorBright+colorWhite, text)
}

// White returns text in white color.
func White(text string) string {
	return colorize(colorWhite, text)
}

// Gray returns text in gray color.
func Gray(text string) string {
	return colorize(colorGray, text)
}

// MediumGray returns text in medium gray (50%) color.
func MediumGray(text string) string {
	return colorize(colorMediumGray, text)
}

// PrintHeader prints a header with bright cyan color.
func PrintHeader(title string) {
	fmt.Printf("# %s\n", BrightCyan(title))
}

// PrintSectionStart prints a section start marker.
func PrintSectionStart(section string) {
	fmt.Printf("%s %s\n", colorize(colorMagenta, "▶"), BrightMagenta(section))
}

// PrintSectionEnd prints a section end marker.
func PrintSectionEnd(section string, success bool) {
	if success {
		fmt.Printf("%s %s\n", colorize(colorGreen, "✓"), BrightGreen(section))
	} else {
		fmt.Printf("%s %s\n", colorize(colorRed, "✗"), BrightRed(section))
	}
}

// PrintPass prints a passing item.
func PrintPass(name string) {
	fmt.Printf(" %s %s\n", colorize(colorGreen, "[✓]"), BrightGreen(name))
}

// PrintFail prints a failed item with error message.
func PrintFail(name, errMsg string) {
	fmt.Printf(" %s %s\n", colorize(colorRed, "[✗]"), BrightRed(name))
	fmt.Printf("    %s %s\n", colorize(colorRed, "→"), colorize(colorRed, errMsg))
}

// PrintInfo prints an information line.
func PrintInfo(key, value string) {
	fmt.Printf(" %s %s %s\n", colorize(colorCyan, "●"), BrightCyan(key+":"), colorize(colorYellow, value))
}

// PrintSuccess prints a success message.
func PrintSuccess(text string) {
	fmt.Printf("%s %s\n", colorize(colorGreen, ">>"), BrightGreen(text))
}

// PrintWarning prints a warning message.
func PrintWarning(text string) {
	fmt.Printf("%s %s\n", colorize(colorYellow, "⚠"), BrightYellow(text))
}
