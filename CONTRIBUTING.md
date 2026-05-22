# Contributing to mdtree

Thanks for your interest in improving mdtree.

## Development setup

Requirements: **Go 1.25+** and **Node.js 20+**.

```bash
git clone https://github.com/SinlinLi/mdtree.git
cd mdtree
./scripts/dev.sh --root ~/some/markdown/folder
```

`dev.sh` starts the Go backend on `:8080` and the Vite dev server on `:5173`.
Open **http://localhost:5173** — it proxies API calls to the backend and
hot-reloads the frontend.

## Project layout

| Path            | Purpose                                            |
| --------------- | -------------------------------------------------- |
| `cmd/mdtree`    | Entry point, CLI and process wiring                |
| `internal/`     | Backend packages (config, logger, auth, files, …)  |
| `web/`          | React + TypeScript frontend, embedded into the binary |
| `docs/`         | Architecture, configuration, API and security docs |

See [`docs/architecture.md`](docs/architecture.md) for how the pieces fit
together.

## Before opening a pull request

```bash
make test    # Go test suite
make lint    # go vet + golangci-lint
make build   # full build, frontend + binary
```

- Keep changes focused and follow the existing style (`gofmt` for Go, the
  Prettier-style 2-space formatting for the frontend).
- Add tests for new behaviour; security-sensitive code (path handling, auth)
  must be covered.
- Update `CHANGELOG.md` under `[Unreleased]`.

## Commit messages

Use clear, imperative messages (`add filename search ranking`, not
`added stuff`). Conventional Commit prefixes (`feat:`, `fix:`, `docs:`) are
welcome but not required.

## Reporting security issues

Please do not file public issues for security vulnerabilities. See
[`docs/security.md`](docs/security.md) for the disclosure process.
