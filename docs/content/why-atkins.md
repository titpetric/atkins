---
title: Why use Atkins?
subtitle: When and why to choose Atkins
layout: page
---

# Why use Atkins?

Atkins fills a specific niche: a command runner that works the same way on your laptop and in CI, with an emphasis on simplicity and YAML-friendly syntax. It's not trying to replace GitHub Actions or be a full CI platform—it's a tool for running tasks locally and in automation.

This page explains when Atkins is a good fit and compares it with related tools.

## When to Choose Atkins

Atkins is a good fit when:

- **You want environment inheritance without boilerplate.** Commands inherit the full shell environment automatically. Variables set in one step are available in the next without extra configuration.

- **You don't need secrets management built into the runner.** Atkins keeps things simple—secrets are handled by your environment or an external tool, not baked into the runner itself.

- **You want YAML-friendly interpolation.** The `${{ }}` syntax doesn't require quoting in YAML and won't collide with bash `${var}` constructs.

- **You want a minimal, fast binary.** Atkins ships as a single ~10MB binary with no runtime dependencies.

- **You need local and CI parity.** Write your pipeline once and run it the same way on your machine and in CI.

- **You want parallel execution with visual progress.** Detach jobs or steps to run concurrently, with a tree-view output that shows what's happening.

- **You want modular, reusable pipelines.** Skills let you compose and conditionally activate groups of jobs across projects.

## Comparison Table

| Feature | Atkins | GitHub Actions | Taskfile | Lefthook |
|---------|--------|----------------|----------|----------|
| Primary use case | Local dev + CI runner | CI/CD platform | Task runner | Git hooks |
| Distributed execution | No [^1] | Yes [^2] | No [^3] | No [^4] |
| Variable interpolation | `${{ var }}` [^5] | `${{ env.VAR }}` [^6] | `{{.Var}}` (Go templates) [^7] | N/A [^8] |
| Shell exec interpolation | `$(cmd)` [^9] | N/A [^10] | `sh: cmd` [^11] | N/A [^12] |
| Secrets management | No | Yes (encrypted) [^13] | No [^14] | No |
| Environment inheritance | Full [^15] | Explicit [^16] | Partial [^17] | Full |
| Parallel execution | Yes (`detach: true`) [^18] | Yes (jobs) [^19] | Yes (`--parallel`) [^20] | Yes (parallel hooks) [^21] |
| Conditional execution | `if:` (expr-lang) [^22] | `if:` (expressions) [^23] | `preconditions:` [^24] | `skip` patterns [^25] |
| File discovery | Auto-discovers config [^26] | Fixed `.github/workflows/` [^27] | `Taskfile.yml` [^28] | `.lefthook.yml` [^29] |
| Dependencies | `depends_on:` | `needs:` | `deps:` | N/A |
| Plugin/extension system | Skills [^30] | Actions marketplace [^31] | Includes [^32] | N/A |
| Output formats | Tree, JSON, YAML | Logs | Text | Text |
| Binary size | ~10MB | N/A (cloud) | ~15MB | ~5MB |
| Shebang support | Yes | No | No | No |
| Stdin pipeline | Yes | No | No | No |

[^1]: Atkins runs locally on a single machine
[^2]: [Using jobs in a workflow](https://docs.github.com/en/actions/using-jobs/using-jobs-in-a-workflow)
[^3]: [Taskfile](https://taskfile.dev/)
[^4]: [Lefthook](https://github.com/evilmartians/lefthook)
[^5]: YAML-compatible, no quoting needed
[^6]: [GitHub Actions expressions](https://docs.github.com/en/actions/learn-github-actions/expressions)
[^7]: Go template syntax, requires quoting in YAML
[^8]: Lefthook uses shell environment variables directly
[^9]: Bash-compatible subshell execution within YAML
[^10]: Use `run:` step output instead
[^11]: [Taskfile dynamic variables](https://taskfile.dev/usage/#dynamic-variables)
[^12]: Uses shell directly
[^13]: [Encrypted secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
[^14]: Relies on environment or external secret management
[^15]: Commands inherit full shell environment automatically
[^16]: [GitHub Actions variables](https://docs.github.com/en/actions/learn-github-actions/variables)
[^17]: [Taskfile environment variables](https://taskfile.dev/usage/#environment-variables)
[^18]: Via `detach: true` on jobs or steps
[^19]: [Using jobs](https://docs.github.com/en/actions/using-jobs)
[^20]: [Running tasks in parallel](https://taskfile.dev/usage/#running-tasks-in-parallel)
[^21]: [Lefthook configuration](https://github.com/evilmartians/lefthook/blob/master/docs/configuration.md)
[^22]: Uses [expr-lang.org](https://expr-lang.org) for condition evaluation
[^23]: [GitHub Actions expressions](https://docs.github.com/en/actions/learn-github-actions/expressions)
[^24]: [Taskfile preconditions](https://taskfile.dev/usage/#preconditions)
[^25]: [Lefthook configuration](https://github.com/evilmartians/lefthook/blob/master/docs/configuration.md)
[^26]: Searches `.atkins.yml`, `atkins.yml` and parent directories
[^27]: Fixed directory structure in repository
[^28]: [Getting started with Taskfile](https://taskfile.dev/usage/#getting-started)
[^29]: [Lefthook](https://github.com/evilmartians/lefthook)
[^30]: Modular pipeline components with conditional activation
[^31]: [Actions marketplace](https://github.com/marketplace?type=actions)
[^32]: [Including other Taskfiles](https://taskfile.dev/usage/#including-other-taskfiles)

## See Also

- [Introduction](./getting-started/introduction) — Overview and quick start
- [Migrating to Atkins](./getting-started/migrating) — Migration guides
