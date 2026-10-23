package storage

import (
	"sort"
	"strings"
)

// shellOperators end a normalized command. A pipeline or a list is
// several commands, and the row groups by the first of them.
var shellOperators = []string{"|", "&&", "||", ";", ">", ">>", "<", "&"}

// Normalize reduces a command to the form the statistics group by, and
// returns it with the tool that leads it.
//
// The form keeps the tool and its subcommands, keeps the flags with
// their values removed, sorts the flags, and drops everything else:
//
//	go test -v ./...                      go test -v
//	go test ./... -v                      go test -v
//	go test -cover -coverprofile=c.out    go test -cover -coverprofile
//	go test --cpuprofile cpu.out          go test --cpuprofile
//	CGO_ENABLED=0 go build -o bin/x .     go build -o
//	sleep 1                               sleep
//	head -20 file.txt                     head
//	git rev-parse HEAD | cut -c1-8        git rev-parse HEAD
//
// Sorting is what makes `go test -v -cover` and `go test -cover -v` one
// row. Dropping operands is what makes a flag's argument disappear with
// no table of which flags take one: `--cpuprofile cpu.out` leaves the
// flag and the file is an operand like any other. A flag that is only
// digits goes with them, so `head -20` and `head -50` count together.
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

		// An assignment ahead of the tool is the environment the command
		// runs with, not the command: `CGO_ENABLED=0 go build` is a go
		// build.
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

// split breaks a command into words the way a shell does. A quoted run
// is one word and keeps the quote, so what it holds is never read as a
// flag or a subcommand however it is spelled.
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

// normalizeFlag strips a flag down to its name: the value it carries,
// the digits it ends with, and the whole flag when nothing is left.
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
