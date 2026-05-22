# Changelog

All notable changes to mdtree are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Markdown-only file tree with lazy-loaded directories.
- Browse and edit markdown files: CodeMirror 6 source editor with a live,
  sanitized preview (edit / split / preview view modes).
- Indexed filename search exposed as a command palette (`Ctrl`/`Cmd` + `P`).
- File operations: create, save, rename, delete, and create directory.
- Password authentication with bcrypt hashing, HTTP-only session cookies and
  login rate limiting.
- Single self-contained binary: the frontend is embedded with `go:embed`.
- Structured, leveled logging (`log/slog`) with console + rotating-file
  output and per-module filtering.
- `/healthz` health check and `/api/stats` runtime metrics.
- `mdtree hash` subcommand to generate a bcrypt password hash. It reads the
  password from stdin when not run on a terminal, for scripted setup.
- `auth.cookie_secure` config option to mark the session cookie Secure when
  serving over HTTPS (including behind a TLS-terminating reverse proxy).

[Unreleased]: https://github.com/SinlinLi/mdtree/commits/main
