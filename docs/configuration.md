# Configuration

mdtree resolves configuration from four sources, each overriding the previous:

1. Built-in defaults
2. A YAML config file (`--config`, default `config.yaml`)
3. `MDTREE_*` environment variables
4. Command-line flags

A missing config file is not an error — defaults and environment variables are
used. Start from [`config.example.yaml`](../config.example.yaml).

## Options

### `server`

| Key    | Default       | Description                                          |
| ------ | ------------- | ---------------------------------------------------- |
| `host` | `127.0.0.1`   | Bind address. Keep it local unless behind a proxy.   |
| `port` | `8080`        | TCP port (1–65535).                                  |

### `root`

| Key    | Default | Description                                               |
| ------ | ------- | --------------------------------------------------------- |
| `root` | `/`     | The directory mdtree may browse and edit. `/` exposes the whole filesystem. Narrow it if you do not need that reach. |

### `auth`

| Key             | Default | Description                                            |
| --------------- | ------- | ------------------------------------------------------ |
| `password_hash` | `""`    | bcrypt hash; generate with `mdtree hash`.              |
| `password`      | `""`    | Optional plaintext password, hashed at startup.        |
| `session_ttl`   | `24h`   | Session lifetime (Go duration string).                 |

If both `password_hash` and `password` are empty, mdtree generates a random
password at startup (system CSPRNG) and prints it once to the console. That
password is not persisted and changes on every restart.

### `log`

| Key           | Default  | Description                                       |
| ------------- | -------- | ------------------------------------------------- |
| `level`       | `info`   | `debug` \| `info` \| `warn` \| `error`.           |
| `dir`         | `./logs` | Directory for rotating log files.                 |
| `console`     | `true`   | Also print human-readable logs to stderr.         |
| `max_backups` | `5`      | Number of rotated log files to keep.              |
| `max_size_mb` | `10`     | Rotate the active log file past this size.        |

The active file is `mdtree.log` (JSON lines). Each run rotates the previous
file, so a run always starts fresh.

### `search`

| Key               | Default                                       | Description                                  |
| ----------------- | --------------------------------------------- | -------------------------------------------- |
| `ignore`          | `.git`, `node_modules`, `.cache`, `vendor`, `.Trash` | Directory names skipped while indexing. |
| `follow_symlinks` | `false`                                       | Follow symlinked directories.                |
| `max_files`       | `200000`                                      | Safety cap on indexed files.                 |

## Environment variables

| Variable               | Overrides              |
| ---------------------- | ---------------------- |
| `MDTREE_HOST`          | `server.host`          |
| `MDTREE_PORT`          | `server.port`          |
| `MDTREE_ROOT`          | `root`                 |
| `MDTREE_PASSWORD`      | `auth.password`        |
| `MDTREE_PASSWORD_HASH` | `auth.password_hash`   |
| `MDTREE_LOG_LEVEL`     | `log.level`            |
| `MDTREE_LOG_DIR`       | `log.dir`              |

## Command-line flags

| Flag          | Description                          |
| ------------- | ------------------------------------ |
| `--config`    | Path to the YAML config file.        |
| `--host`      | Override the server host.            |
| `--port`      | Override the server port.            |
| `--root`      | Override the browsable root.         |
| `--log-level` | Override the log level.              |

## Subcommands

| Command          | Description                                              |
| ---------------- | -------------------------------------------------------- |
| `mdtree hash`    | Prompt for a password and print its bcrypt hash.         |
| `mdtree version` | Print the version.                                       |
