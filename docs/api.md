# API reference

mdtree exposes a small JSON API under `/api`. All responses are
`application/json`. Errors use `{"error": "message"}` with an appropriate HTTP
status.

Authentication is by session cookie (`mdtree_session`, HTTP-only,
`SameSite=Strict`). Every endpoint except the health check and the auth
endpoints requires a valid session; without one they return `401`.

## Authentication

### `POST /api/auth/login`

Body: `{"password": "..."}`. On success sets the session cookie.

```json
{ "ok": true }
```

`401` on a wrong password, `429` when rate limited (8 failures per minute per
client IP).

### `POST /api/auth/logout`

Invalidates the current session and clears the cookie.

### `GET /api/auth/status`

```json
{ "authenticated": true }
```

## Files

Paths are absolute server paths and must resolve inside the configured `root`.
Paths outside it return `403`; non-markdown files return `400`.

### `GET /api/tree?path=<dir>`

Lists one directory: every subdirectory plus markdown files only. `path`
defaults to `root`.

```json
{
  "path": "/srv/docs",
  "parent": "/srv",
  "entries": [
    { "name": "guide.md", "path": "/srv/docs/guide.md", "type": "file", "size": 1234, "modTime": "2026-05-22T10:00:00Z" }
  ]
}
```

### `GET /api/file?path=<file.md>`

Returns a markdown file with its content.

```json
{ "path": "/srv/docs/guide.md", "name": "guide.md", "size": 1234, "modTime": "...", "content": "# Guide\n..." }
```

### `POST /api/file`

Create a file. Body: `{"path": "...", "content": "..."}`. `201` on success,
`409` if it already exists.

### `PUT /api/file`

Save (overwrite) an existing file. Body: `{"path": "...", "content": "..."}`.
`404` if the file does not exist.

### `DELETE /api/file?path=<file.md>`

Delete a markdown file. Directories and non-markdown files cannot be deleted.

### `POST /api/file/rename`

Body: `{"from": "...", "to": "..."}`. `409` if the destination exists.

### `POST /api/dir`

Body: `{"path": "..."}`. Creates a directory (and any missing parents).

## Search

### `GET /api/search?q=<query>&limit=<n>`

Ranked filename search over the in-memory index. `limit` defaults to 50.

```json
{
  "query": "guide",
  "count": 1,
  "results": [
    { "name": "guide.md", "path": "/srv/docs/guide.md", "score": 842 }
  ]
}
```

### `POST /api/search/reindex`

Rebuilds the index from disk.

```json
{ "files": 1280, "durationMs": 41.7 }
```

## Observability

### `GET /healthz`

Unauthenticated liveness check: `{"status":"ok"}`.

### `GET /api/stats`

Runtime metrics and index statistics.

```json
{
  "metrics": { "uptimeSeconds": 3600, "requests": 420, "requestErrors": 0, "avgLatencyMs": 1.8, "fileReads": 50, "fileWrites": 12, "searches": 30, "indexedFiles": 1280, "lastIndexBuildMs": 41.7 },
  "index": { "files": 1280, "builtAt": "2026-05-22T10:00:00Z", "buildMillis": 41.7 },
  "sessions": 1
}
```

## Status codes

| Code | Meaning                                            |
| ---- | -------------------------------------------------- |
| 400  | Invalid path, or not a markdown file               |
| 401  | No valid session                                   |
| 403  | Path resolves outside the configured root          |
| 404  | File or directory not found                        |
| 409  | Target already exists                              |
| 413  | File exceeds the 10 MiB limit                      |
| 429  | Login rate limit exceeded                          |
