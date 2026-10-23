# Package ./colors

```go
import (
	"github.com/titpetric/atkins/colors"
)
```

Package `colors` wraps text in ANSI colour escapes and measures text that
carries them. It leaves the choice to colour to the caller, with
StdoutIsTerminal as the usual input to it.

## Function symbols

- `func BrightCyan (text string) string`
- `func BrightGreen (text string) string`
- `func BrightMagenta (text string) string`
- `func BrightOrange (text string) string`
- `func BrightRed (text string) string`
- `func BrightWhite (text string) string`
- `func BrightYellow (text string) string`
- `func Cyan (text string) string`
- `func Dim (text string) string`
- `func Gray (text string) string`
- `func Green (text string) string`
- `func MediumGray (text string) string`
- `func StdoutIsTerminal () bool`
- `func StripANSI (in string) string`
- `func VisualLength (s string) int`
- `func White (text string) string`

### `BrightCyan`

`BrightCyan` returns text in bright cyan color.

```go
func BrightCyan(text string) string
```

### `BrightGreen`

`BrightGreen` returns text in bright green color.

```go
func BrightGreen(text string) string
```

### `BrightMagenta`

`BrightMagenta` returns text in bright magenta color.

```go
func BrightMagenta(text string) string
```

### `BrightOrange`

`BrightOrange` returns text in bright orange color.

```go
func BrightOrange(text string) string
```

### `BrightRed`

`BrightRed` returns text in bright red color.

```go
func BrightRed(text string) string
```

### `BrightWhite`

`BrightWhite` returns text in bright white color.

```go
func BrightWhite(text string) string
```

### `BrightYellow`

`BrightYellow` returns text in bright yellow color.

```go
func BrightYellow(text string) string
```

### `Cyan`

`Cyan` returns text in cyan color.

```go
func Cyan(text string) string
```

### `Dim`

`Dim` returns text in dim color.

```go
func Dim(text string) string
```

### `Gray`

`Gray` returns text in gray color.

```go
func Gray(text string) string
```

### `Green`

`Green` returns text in green color.

```go
func Green(text string) string
```

### `MediumGray`

`MediumGray` returns text in medium gray (50%) color.

```go
func MediumGray(text string) string
```

### `StdoutIsTerminal`

`StdoutIsTerminal` reports whether stdout is a terminal, checked once per process.

```go
func StdoutIsTerminal() bool
```

### `StripANSI`

`StripANSI` removes all ANSI escape sequences from a string.

```go
func StripANSI(in string) string
```

### `VisualLength`

`VisualLength` returns the visual length of a string (excluding ANSI sequences).

```go
func VisualLength(s string) int
```

### `White`

`White` returns text in white color.

```go
func White(text string) string
```
