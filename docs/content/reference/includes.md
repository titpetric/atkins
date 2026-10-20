---
title: Includes
subtitle: File inclusion reference
layout: page
---

`include:` reads one or more files into `vars`, at the pipeline, job or step level. It does not merge jobs: Atkins ignores an included file's `jobs:` section, if it has one. To share jobs across files, use [Skills](../usage/skills).

## Basic Include

@tabs
@file "Main" includes/main.yml
@file "build.yml" includes/ci/build.yml
@file "test.yml" includes/ci/test.yml

![Includes](./includes/main.png)

`ci/build.yml` and `ci/test.yml` are plain YAML documents of variables, not pipelines. Their top-level keys land in `vars`, exactly as if `main.yml` declared them directly.

## Merge Order

A later file in the list overrides an earlier one, and the block's own `vars:` overrides every included file, whichever order they appear in:

```yaml
include:
  - ci/build.yml   # tag: v1.2.3
  - ci/test.yml    # tag: v1.2.3-rc (wins over ci/build.yml)

vars:
  tag: v2.0.0        # wins over both included files
```

## Relative Paths

Atkins resolves a path in `include:` from the current working directory of the `atkins` process, not from the file that declares it. A pipeline that includes a file next to itself only works when invoked from that directory; write the include relative to the directory you run the pipeline from, or resolve it with `$(...)` first.

No glob expansion happens: Atkins reads `include: ci/*.yml` as a literal filename and fails with "no such file or directory". List every file `include:` should read.

## Environment Includes

`env: include:` does the same for `env`, but reads a dotenv file (`KEY=value` lines) in place of YAML:

```yaml
env:
  include: .env.production
  vars:
    LOG_LEVEL: debug   # wins over .env.production
```

## See Also

- [Pipeline](./pipeline) - Pipeline configuration
- [Variables](./variables) - Variable interpolation
- [Skills](../usage/skills) - Sharing jobs across files
