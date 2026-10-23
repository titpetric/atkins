package storage

import (
	"sort"
	"strings"
)

// shellOperators end a normalized command. A pipeline or a list holds
// more than one command, and the row groups by the first.
var shellOperators = []string{"|", "&&", "||", ";", ">", ">>", "<", "&"}

// Normalize returns two empty strings for a command with no tool, else
// its stored form and tool, as the package documentation lists them.
func Normalize(command string) (normalized, tool string) {
	words := split(command)

	var path, flags []string
	seen := map[string]bool{}
	operand := false

	for _, word := range words {
		if isOperator(word) {
			break
		}

		if strings.HasPrefix(word, "-") {
			flag := normalizeFlag(word)
			if flag != "" && !seen[flag] {
				seen[flag] = true
				flags = append(flags, flag)
			}
			operand = true
			continue
		}

		// An assignment ahead of the tool sets the environment of the
		// command: `CGO_ENABLED=0 go build` counts as a go build.
		if len(path) == 0 && isAssignment(word) {
			continue
		}

		if operand || !isPathWord(word) {
			operand = true
			continue
		}

		path = append(path, word)
	}

	if len(path) == 0 {
		return "", ""
	}

	sort.Strings(flags)

	return strings.Join(append(path, flags...), " "), path[0]
}

// split returns the words of a command the way a shell breaks them. A
// quoted run makes one word that keeps the quote, so its text never
// counts as a flag or a subcommand.
func split(command string) []string {
	var (
		words  []string
		word   strings.Builder
		quote  rune
		quoted bool
	)

	flush := func() {
		if word.Len() == 0 && !quoted {
			return
		}
		text := word.String()
		if quoted {
			text = `"` + text
		}
		words = append(words, text)
		word.Reset()
		quoted = false
	}

	for _, r := range command {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			word.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
			quoted = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			word.WriteRune(r)
		}
	}
	flush()

	return words
}

// isOperator reports whether a word ends the command.
func isOperator(word string) bool {
	for _, operator := range shellOperators {
		if word == operator {
			return true
		}
	}
	return false
}

// normalizeFlag returns the name of a flag without its value or trailing
// digits, and an empty string when no name remains.
func normalizeFlag(word string) string {
	flag := word
	if at := strings.IndexByte(flag, '='); at >= 0 {
		flag = flag[:at]
	}
	flag = strings.TrimRight(flag, "0123456789")
	if strings.Trim(flag, "-") == "" {
		return ""
	}
	return flag
}

// isAssignment reports whether a word sets an environment variable.
func isAssignment(word string) bool {
	at := strings.IndexByte(word, '=')
	if at <= 0 {
		return false
	}
	for index, r := range word[:at] {
		upper := r >= 'A' && r <= 'Z'
		lower := r >= 'a' && r <= 'z'
		digit := r >= '0' && r <= '9'
		if !upper && !lower && r != '_' && !(digit && index > 0) {
			return false
		}
	}
	return true
}

// isPathWord reports whether a word belongs to the command's name: the
// tool and the subcommands after it. A path, a number, a glob and
// anything holding a shell character is an operand.
func isPathWord(word string) bool {
	if word == "" {
		return false
	}
	if strings.ContainsAny(word, "/\\*?$`(){}[]'\" \t=") {
		return false
	}
	if strings.HasPrefix(word, ".") {
		return false
	}
	return strings.Trim(word, "0123456789.,:") != ""
}

// keyWords caps a command key: the tool and two subcommands, which is
// `docker compose up` and `git rev-parse HEAD` and never a list of
// operands.
const keyWords = 3

// builtins are the shell's own words a step runs to set itself up. They
// cost nothing worth a row.
var builtins = []string{"set", "cd", "export", "echo", "test", "[", "true", "false", ":"}

// IsBuiltin reports whether a tool is a shell builtin, which the
// statistics neither record nor report.
func IsBuiltin(tool string) bool {
	for _, builtin := range builtins {
		if tool == builtin {
			return true
		}
	}
	return false
}

// Key returns the report key of a command and its tool, or two empty
// strings for a builtin or a command with no tool.
func Key(command string) (key, tool string) {
	var words []string

	for _, word := range split(command) {
		if isOperator(word) || strings.HasPrefix(word, "-") || len(words) == keyWords {
			break
		}
		if len(words) == 0 && isAssignment(word) {
			continue
		}
		if !isPathWord(word) || strings.Contains(word, ".") {
			break
		}
		words = append(words, word)
	}

	if len(words) == 0 || IsBuiltin(words[0]) {
		return "", ""
	}

	return strings.Join(words, " "), words[0]
}
