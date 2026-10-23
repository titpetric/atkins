package colors_test

import (
	"os"
	"testing"

	"golang.org/x/term"

	"github.com/titpetric/atkins/colors"
	"github.com/titpetric/atkins/internal/assert"
)

func TestStdoutIsTerminal(t *testing.T) {
	t.Parallel()

	want := term.IsTerminal(int(os.Stdout.Fd()))
	assert.Equal(t, want, colors.StdoutIsTerminal())
	assert.Equal(t, want, colors.StdoutIsTerminal())
}

func TestColors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		render func(string) string
		prefix string
	}{
		{"BrightGreen", colors.BrightGreen, "\033[1m\033[32m"},
		{"Green", colors.Green, "\033[32m"},
		{"BrightYellow", colors.BrightYellow, "\033[1m\033[33m"},
		{"BrightOrange", colors.BrightOrange, "\033[38;5;208m"},
		{"BrightCyan", colors.BrightCyan, "\033[1m\033[36m"},
		{"Cyan", colors.Cyan, "\033[36m"},
		{"BrightMagenta", colors.BrightMagenta, "\033[1m\033[35m"},
		{"BrightRed", colors.BrightRed, "\033[1m\033[31m"},
		{"Dim", colors.Dim, "\033[2m"},
		{"BrightWhite", colors.BrightWhite, "\033[1m\033[37m"},
		{"White", colors.White, "\033[37m"},
		{"Gray", colors.Gray, "\033[90m"},
		{"MediumGray", colors.MediumGray, "\033[38;5;244m"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rendered := tc.render("text")
			assert.Equal(t, tc.prefix+"text\033[0m", rendered)
			assert.Equal(t, "text", colors.StripANSI(rendered))
			assert.Equal(t, 4, colors.VisualLength(rendered))
		})
	}
}
