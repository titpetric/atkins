package model

// Label is the display label of a step or command. NewLabel starts one and
// the With* builders customize it. String is the plain name used for node
// names and comparisons; ForDisplay adds the colour and status for rendering.
type Label struct {
	Text       string              // The display text (e.g., "docker compose up" or "run: splint fix ./...")
	Type       string              // The operation kind: "task", "run", "cmd", "cmds"
	ShowPrefix bool                // Whether to display the type prefix (e.g., "run:")
	Status     string              // Optional status indicator (e.g., "●", "✓", "✗")
	Color      func(string) string // Optional color function to apply to the label
}

// NewLabel returns a label with the given text and type.
func NewLabel(text, labelType string) *Label {
	return &Label{
		Text: text,
		Type: labelType,
	}
}

// WithPrefix sets whether to show the type prefix in output.
func (l *Label) WithPrefix(show bool) *Label {
	l.ShowPrefix = show
	return l
}

// WithStatus sets the status indicator to display.
func (l *Label) WithStatus(status string) *Label {
	l.Status = status
	return l
}

// WithColor sets a color function to apply to the label.
func (l *Label) WithColor(colorFn func(string) string) *Label {
	l.Color = colorFn
	return l
}

// String returns the label text, with the type in front when ShowPrefix holds.
func (l *Label) String() string {
	text := l.Text
	if l.ShowPrefix {
		text = l.Type + ": " + text
	}
	return text
}

// ForDisplay returns String with the colour applied and the status appended.
func (l *Label) ForDisplay() string {
	text := l.String()

	// Apply color if provided
	if l.Color != nil {
		text = l.Color(text)
	}

	// Append status if provided
	if l.Status != "" {
		text = text + " " + l.Status
	}

	return text
}
