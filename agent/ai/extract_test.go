package ai

import "testing"

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{
			name:  "plain object",
			input: `{"message": "done"}`,
			want:  `{"message": "done"}`,
			ok:    true,
		},
		{
			name:  "surrounded by prose",
			input: "Sure, here you go:\n" + `{"cmds": ["atkins", "atkins release"]}` + "\nLet me know if you need more.",
			want:  `{"cmds": ["atkins", "atkins release"]}`,
			ok:    true,
		},
		{
			name:  "wrapped in a code fence",
			input: "```json\n" + `{"cmds": ["atkins release"]}` + "\n```",
			want:  `{"cmds": ["atkins release"]}`,
			ok:    true,
		},
		{
			name:  "brace inside a string value doesn't break balancing",
			input: `{"message": "added job { for build }"}`,
			want:  `{"message": "added job { for build }"}`,
			ok:    true,
		},
		{
			name:  "no object present",
			input: "no can do",
			ok:    false,
		},
		{
			name:  "unbalanced object",
			input: `{"cmds": ["atkins"]`,
			ok:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractJSON(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
