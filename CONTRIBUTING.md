# Contributing to PiHolster

Thank you for your interest in contributing! Please read this guide before opening a pull request.

## Change process

All significant changes — new features, breaking API changes, and architectural decisions — must follow the process described in [docs/CHANGE-PROCESS.md](docs/CHANGE-PROCESS.md).

In short:

1. Open a GitHub Issue describing the problem or proposal before writing code.
2. For architectural decisions, write an ADR in `docs/` following the existing template.
3. Fork the repo and create a branch from `develop`.
4. Open a pull request when your change is ready for review.

## Branch naming

| Type | Pattern | Example |
|------|---------|---------|
| New feature | `feature/<short-description>` | `feature/arp-alerts` |
| Bug fix | `fix/<short-description>` | `fix/dns-leak-on-restart` |
| Hotfix (production) | `hotfix/<short-description>` | `hotfix/auth-bypass` |
| Chore / refactor | `chore/<short-description>` | `chore/update-blocklists` |

Branch names must be lowercase and use hyphens, not underscores.

## Pull request requirements

- Target branch: `develop` (never `main` directly, except hotfixes)
- CI must pass: `go vet`, `go test -race`, `golangci-lint`, `pnpm lint`, `pnpm build`
- At least one maintainer approval is required before merge
- Squash merge is preferred; keep commit history clean
- Update `CHANGELOG.md` (or reference the issue) in your PR description
- For UI changes, include a screenshot or screen recording

## Development setup

```bash
# Prerequisites: Go 1.22+, Node 20+, pnpm 9+
make dev       # starts backend :8080 and frontend :5173 in parallel
make test      # runs all tests
make lint      # runs all linters
```

## Code style

- Go: `gofmt` + `golangci-lint` (config in `.golangci.yml`)
- Svelte/JS: ESLint via `pnpm --filter web lint`
- Commit messages: Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, etc.)

## Reporting security vulnerabilities

Do **not** open a public GitHub Issue for security vulnerabilities. Email the maintainers directly. See [docs/SECURITY.md](docs/SECURITY.md) if available.
