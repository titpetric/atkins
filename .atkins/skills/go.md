`atkins go` runs the whole lifecycle: generate, fmt, lint, test, build.

- `atkins go:fmt` - format the sources. Run `atkins go:fmt:revise` after it and never before it: goimports re-sorts the groups the reviser arranged.
- `atkins go:test` - run the tests and write the coverage profile `pkg.cov`.
