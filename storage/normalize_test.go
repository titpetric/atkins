package storage_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
		tool    string
	}{
		{name: "a bare tool", command: "pwd", want: "pwd", tool: "pwd"},
		{name: "a subcommand", command: "go build", want: "go build", tool: "go"},
		{name: "operands drop", command: "go test ./...", want: "go test", tool: "go"},
		{name: "a flag stays", command: "go test -v ./...", want: "go test -v", tool: "go"},
		{
			name:    "flag order does not matter",
			command: "go test -cover -v ./...",
			want:    "go test -cover -v",
			tool:    "go",
		},
		{
			name:    "the other order is the same row",
			command: "go test -v -cover ./...",
			want:    "go test -cover -v",
			tool:    "go",
		},
		{
			name:    "an inline value drops",
			command: "go test -coverprofile=coverage.out ./...",
			want:    "go test -coverprofile",
			tool:    "go",
		},
		{
			name:    "a separate value drops with the operands",
			command: "go test --cpuprofile cpu.out ./...",
			want:    "go test --cpuprofile",
			tool:    "go",
		},
		{name: "a numeric flag drops", command: "head -20 README.md", want: "head", tool: "head"},
		{name: "a numeric operand drops", command: "sleep 1", want: "sleep", tool: "sleep"},
		{name: "and so does the next one", command: "sleep 30", want: "sleep", tool: "sleep"},
		{
			name:    "an environment assignment is not the tool",
			command: "CGO_ENABLED=0 go build -o bin/atkins .",
			want:    "go build -o",
			tool:    "go",
		},
		{
			name:    "a repeated flag counts once",
			command: "docker run -v a:a -v b:b image",
			want:    "docker run -v",
			tool:    "docker",
		},
		{
			name:    "a pipeline is its first command",
			command: "git rev-parse HEAD | cut -c1-8",
			want:    "git rev-parse HEAD",
			tool:    "git",
		},
		{
			name:    "a quoted operand is one word and drops",
			command: `echo "deploying to staging"`,
			want:    "echo",
			tool:    "echo",
		},
		{
			name:    "a quoted operand that looks like a flag still drops",
			command: `echo "-v --help"`,
			want:    "echo",
			tool:    "echo",
		},
		{
			name:    "a path is never a subcommand",
			command: "./bin/deploy --dry-run",
			want:    "",
			tool:    "",
		},
		{name: "nothing at all", command: "", want: "", tool: ""},
		{name: "flags without a tool", command: "-v -x", want: "", tool: ""},
		{
			name:    "a newline in a step is whitespace",
			command: "go vet\n./...",
			want:    "go vet",
			tool:    "go",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, tool := storage.Normalize(test.command)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.tool, tool)
		})
	}
}

func TestKey(t *testing.T) {
	tests := []struct {
		command string
		want    string
		tool    string
	}{
		{command: "go test ./...", want: "go test", tool: "go"},
		{command: "go test -v -cover ./...", want: "go test", tool: "go"},
		{command: "docker build -q -t app .", want: "docker build", tool: "docker"},
		{command: "docker compose up -d", want: "docker compose up", tool: "docker"},
		{command: "docker compose build", want: "docker compose build", tool: "docker"},
		{command: "git clone --depth 1 https://example.com/x.git", want: "git clone", tool: "git"},
		{command: "git rev-parse HEAD | cut -c1-8", want: "git rev-parse HEAD", tool: "git"},
		{command: "kubectl get pods web-1 -o yaml", want: "kubectl get pods", tool: "kubectl"},
		{command: "cat README.md", want: "cat", tool: "cat"},
		{command: "npm run build", want: "npm run build", tool: "npm"},
		{command: "CGO_ENABLED=0 go build -o bin/x .", want: "go build", tool: "go"},
		{command: "sleep 30", want: "sleep", tool: "sleep"},
		{command: "make VERSION=1 all", want: "make", tool: "make"},
		{command: "echo $HOME", want: "", tool: ""},
		{command: "set -eu", want: "", tool: ""},
		{command: "cd /src && go build", want: "", tool: ""},
		{command: "[ -f x ]", want: "", tool: ""},
		{command: "true", want: "", tool: ""},
		{command: "./bin/deploy --dry-run", want: "", tool: ""},
		{command: "", want: "", tool: ""},
	}

	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			t.Parallel()
			got, tool := storage.Key(test.command)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.tool, tool)

			// A normalized command keys the same as the command it came
			// from, so old rows group by the key.
			normalized, _ := storage.Normalize(test.command)
			again, _ := storage.Key(normalized)
			assert.Equal(t, test.want, again)
		})
	}
}

func TestIsBuiltin(t *testing.T) {
	t.Parallel()
	assert.True(t, storage.IsBuiltin("set"))
	assert.True(t, storage.IsBuiltin(":"))
	assert.False(t, storage.IsBuiltin("go"))
}
