---
title: Installation
subtitle: How to install Atkins
layout: page
---

# Installing Atkins

There are three methods to install Atkins depending on your environment.

## From Source (Go)

If you have Go installed, this is the simplest method:

```bash
go install github.com/titpetric/atkins@latest
```

## Binary Release

1. Navigate to the [Releases page](https://github.com/titpetric/atkins/releases)
2. Download the binary for your platform (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
3. Install to your PATH:

```bash
# Example for Linux amd64
curl -L https://github.com/titpetric/atkins/releases/latest/download/atkins-linux-amd64 -o /usr/local/bin/atkins
chmod +x /usr/local/bin/atkins
```

## Docker

Atkins is available as a Docker image. Add it to your Dockerfile:

```dockerfile
# Copy atkins from the official image
COPY --from=titpetric/atkins:latest /usr/local/bin/atkins /usr/local/bin/atkins
```

Or use it directly:

```bash
docker run --rm -v $PWD:/app titpetric/atkins -l
```

## Verify Installation

After installation, verify with:

```bash
# Print version
atkins -v

# List tasks in current directory
atkins -l

# Run default task
atkins
```

## Shebang Support (Linux/macOS)

On Unix systems, you can make pipeline files directly executable:

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
