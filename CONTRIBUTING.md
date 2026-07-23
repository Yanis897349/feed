# Contributing

Thanks for helping improve InvestingLive News. Small, focused changes are
welcome, including bug fixes, accessibility improvements, tests, and docs.

## Local setup

1. Install the pinned Go toolchain with `mise install`, or use Go 1.25+.
2. Run the app with `mise run news` or `go run .`.
3. Run the full local check with `mise run check` before opening a pull request.

The individual checks are also available:

```shell
mise run format
mise run test
mise run vet
mise run build
```

## Pull requests

- Keep each change scoped to one concern.
- Add or update tests for behavior changes.
- Update the README when user-facing behavior changes.
- Do not commit build artifacts, coverage reports, credentials, or editor state.
- Use clear commit and pull request descriptions that explain both the change
  and its motivation.

CI checks formatting, static analysis, tests, and cross-platform builds.

## Reporting problems

Use the issue templates for reproducible bugs and feature proposals. Please
report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).
