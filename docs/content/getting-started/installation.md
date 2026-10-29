---
title: Installation
subtitle: How to install Atkins
layout: page
---

Atkins ships as a single binary with no runtime dependencies. You can install it from source using Go, download a pre-built binary, or copy it from the official Docker image. Choose whichever method fits your environment.

## From Source (Go)

If you have Go 1.27+ installed, this is the simplest method:

```bash
go install github.com/titpetric/atkins@latest
```

`go install` puts the binary in your `$GOPATH/bin` directory.

## Binary Release

Pre-built binaries are available for Linux amd64:

1. Navigate to the [Releases page](https://github.com/titpetric/atkins/releases)
2. Download the binary for your platform
3. Install to your PATH:

```bash
# Example for Linux amd64
curl -L https://github.com/titpetric/atkins/releases/latest/download/atkins-linux-amd64 -o /usr/local/bin/atkins
chmod +x /usr/local/bin/atkins
```

## Docker

Atkins is available as `titpetric/atkins:latest`. You can copy the binary into your own images or run it directly.

Add it to your Dockerfile:

```dockerfile
COPY --from=titpetric/atkins:latest /usr/local/bin/atkins /usr/local/bin/atkins
```

Or run it directly:

```bash
docker run --rm -v $PWD:/app titpetric/atkins -l
```

The image builds on [alpine](https://hub.docker.com/_/alpine) in [docker/Dockerfile](https://github.com/titpetric/atkins/blob/main/docker/Dockerfile).

## Verify Installation

After installation, verify Atkins is working:

```bash
# Print version
atkins -v

# List tasks in current directory
atkins -l

# Run default task
atkins
```

## Shebang Support (Linux/macOS)

On Unix systems, a shebang line makes a pipeline file directly executable:

```yaml
#!/usr/bin/env atkins
name: My Script

tasks:
  default:
    steps:
      - run: echo "Hello from executable pipeline!"
```

Then:

```bash
chmod +x my-pipeline.yml
./my-pipeline.yml
```

Atkins strips the shebang line before parsing. The file stays valid YAML for other tools.

## What a run records

Atkins records a row per job and per command of every run in `.atkins/atkins.db`, creating the file on the first run. The database is a pure Go SQLite, and the binary stays one static file.

```bash
atkins --stats
```

See [Statistics](../usage/stats) for the tables, the normalised command and how to turn recording off.

## Next Steps

- [Configuration](../usage/configuration) - Learn the pipeline format
- [CLI Flags](../usage/cli-flags) - Command-line reference
- [Statistics](../usage/stats) - Where the time goes
