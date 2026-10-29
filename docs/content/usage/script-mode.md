---
title: Script Mode
subtitle: Executable pipelines and stdin input
layout: page
---

Atkins runs pipelines as executable scripts or reads them from stdin. This lets you treat pipeline files like shell scripts: make them executable, pass arguments, and integrate them into larger toolchains.

## Shebang Execution

On Linux and macOS, a shebang line makes a pipeline file directly executable:

@tabs
@file "Shebang" script-mode/shebang.yml

![Shebang](./script-mode/shebang.png)

```bash
chmod +x script.yml
./script.yml
```

Atkins strips the shebang line before parsing. The file stays valid YAML for other tools.

## Stdin Input

Atkins also reads a pipeline from stdin:

```bash
cat pipeline.yml | atkins
```

Or with a here-doc:

```bash
atkins <<EOF
tasks:
  default:
    steps:
      - run: echo "From stdin"
EOF
```

This is useful for dynamically generated pipelines or quick one-off runs.

## Positional Arguments

Atkins also takes a file path as a positional argument:

```bash
atkins ./ci/build.yml
```

This matches:

```bash
atkins -f ./ci/build.yml
```

## Combining with Flags

All standard [CLI flags](./cli-flags) work with script mode:

```bash
# Show final tree only
./script.yml --final

# List jobs from an executable pipeline
./script.yml -l

# JSON output from stdin
cat pipeline.yml | atkins --json
```

## See Also

- [CLI Flags](./cli-flags) - Command-line options
- [Automation](./automation) - JSON/YAML output details
