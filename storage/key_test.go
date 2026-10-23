package storage_test

import (
	"testing"

	"github.com/titpetric/atkins/internal/assert"
	"github.com/titpetric/atkins/storage"
)

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
			// from, which is what lets old rows group by the key.
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
