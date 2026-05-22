# Security model

mdtree is built to **browse and edit markdown files anywhere the server
process can reach**. That reach is the feature — and the risk. Read this
before exposing mdtree beyond `localhost`.

## Threat model

mdtree assumes:

- The **password is the only gate**. Anyone who authenticates can read and
  write every markdown file under `root` with the process's permissions.
- The **process privileges define the blast radius**. mdtree does not drop
  privileges. If it runs as `root` with `root: "/"`, an authenticated user can
  edit any markdown file on the host.
- The **network is hostile**. Without TLS, the session cookie and password are
  exposed in transit.

## What mdtree does

- **Password authentication.** Passwords are stored only as bcrypt hashes.
  mdtree never stores or logs the plaintext. The model never generates the
  password — `mdtree hash` takes yours, or the system CSPRNG generates one.
- **Sessions.** Login issues a 256-bit CSPRNG token in an HTTP-only,
  `SameSite=Strict` cookie (also `Secure` when TLS is configured). Sessions
  live in memory and expire after `session_ttl`.
- **Login rate limiting.** Eight failed attempts per minute per client IP are
  then rejected with `429`, slowing brute-force attempts.
- **Path confinement.** Every requested path is cleaned, made absolute, and
  verified to be inside `root`. Symlinks that escape `root` are rejected
  unless `follow_symlinks` is enabled. `..` traversal cannot escape.
- **Markdown-only writes.** Create, save, rename and delete operate only on
  markdown files; mdtree will not delete arbitrary files or directories.
- **CSRF.** The `SameSite=Strict` session cookie is not sent on cross-site
  requests, so a third-party page cannot drive the API as a logged-in user.
- **Output safety.** The preview renderer disables raw HTML in markdown and
  sanitizes the result with DOMPurify before it reaches the DOM.
- **Size limits.** Files above 10 MiB are rejected; request bodies are capped.

## Hardening checklist

When deploying mdtree somewhere reachable:

1. **Run as a dedicated, least-privilege user.** Create a `mdtree` user that
   owns only the files it should edit. Never run it as `root`.
2. **Scope `root`.** If you only need `/srv/docs`, set `root: /srv/docs`
   instead of `/`. Smaller reach, smaller blast radius.
3. **Terminate TLS in front of it.** Keep `host: 127.0.0.1` and put nginx,
   Caddy, or another HTTPS reverse proxy in front. mdtree honours
   `X-Forwarded-For` for accurate rate-limiting and logs.
4. **Use a strong password.** Generate the hash with `mdtree hash` and a long,
   unique passphrase. Set `password_hash`, not `password`.
5. **Keep `follow_symlinks: false`** unless you specifically need it.
6. **Watch the logs.** Failed logins are logged at `WARN` with the client IP.

## Out of scope (today)

mdtree v1 deliberately does not include: multi-user accounts or per-user
permissions, an audit trail of edits, file versioning, or encryption at rest.
Sessions are in memory, so a restart logs everyone out.

## Reporting a vulnerability

Please do not open a public issue for a security vulnerability. Instead, email
the maintainers (see the repository's contact details) with a description and
reproduction steps. You will receive an acknowledgement, and we ask for a
reasonable disclosure window before public discussion.
