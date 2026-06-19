# Flotio CLI

> Command-line tool for managing [Flotio](https://flotio.ovh) cloud infrastructure — the Flutter CI/CD platform for building, signing, and releasing Flutter apps.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/flotio-dev/cli?display_name=tag&sort=semver)](https://github.com/flotio-dev/cli/releases)
[![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macOS%20%7C%20windows-blue)](#installation)
[![License](https://img.shields.io/badge/license-TBD-lightgrey)](#license)

`flotio` lets you authenticate, manage projects, trigger and watch builds, upload signing keys, publish to Google Play, and manage environment variables — all from the terminal.

---

## Features

- **Authentication** — log in with email/password; tokens are cached locally and reused automatically.
- **Project management** — create, list, inspect, update, and delete Flotio projects, including full build configuration.
- **Builds** — start builds, stream/list logs, download artifacts, cancel, or delete them.
- **Releases** — publish builds to Google Play and audit releases. *(new)*
- **Build cache** — purge, inspect metrics, and list cache entries. *(new)*
- **Environment** — manage environment variables **and** files (with `@file` upload syntax and base64 encoding).
- **Signing & publishing** — upload Android keystores and Google Play service-account credentials.
- **GitHub integration** — connect an installation, browse repos, and inspect repo trees.
- **Flutter SDKs** — list available Flutter versions on the platform.
- **Project context** — `flotio init` marks a directory with `.flotio.yaml` so most commands infer the project automatically.
- **Self-diagnostics & updates** — `flotio doctor` checks auth/config/connectivity, `flotio update` self-upgrades from GitHub Releases.

---

## Installation

### Option 1 — Install script (recommended)

**macOS / Linux:**

```sh
curl -fsSL https://raw.githubusercontent.com/flotio-dev/cli/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/flotio-dev/cli/main/install.ps1 | iex
```

The script detects your OS/architecture, downloads the latest release, and installs the `flotio` binary to `/usr/local/bin` (or `~/.local/bin` if that isn't writable).

### Option 2 — `go install`

If you have the Go toolchain (1.26+):

```sh
go install github.com/flotio-dev/cli@latest
```

This places the binary in `$GOBIN` (or `$GOPATH/bin`).

### Option 3 — Build from source

This repo uses [devenv](https://devenv.sh/) (nix-based dev shell). The `devenv.nix` file lives in the **parent** directory, so builds run from there:

```sh
git clone https://github.com/flotio-dev/cli.git
cd cli/..
devenv shell make build      # builds bin/flotio with the current version
```

See [Development](#development) below for the available `make` targets. You can also run `flotio update` later to upgrade any install method.

---

## Quick Start

```sh
# 1. Log in
flotio login -e you@example.com -p yourpassword

# 2. (Optional) point at a different API host
flotio configure --host http://localhost:8080

# 3. Mark this directory as a Flotio project (interactive)
flotio init

# 4. List your projects and available Flutter versions
flotio project list
flotio flutter versions

# 5. Kick off a release build and watch the logs
flotio build start --branch main --platform android --mode release
flotio build logs <build-id>

# 6. Check that everything is healthy
flotio doctor
```

> **Tip:** after `flotio init`, commands that need a project ID (e.g. `flotio build list`, `flotio build start`) read it from `.flotio.yaml` automatically.

---

## Commands Reference

Run `flotio --help` or `flotio <command> --help` for full details on any command.

### Global flags

| Flag | Description |
| --- | --- |
| `--host <url>` | API host (default `api.flotio.ovh`; accepts `scheme://host`, e.g. `http://localhost:8080`) |
| `--config <path>` | Path to config file (default `~/.flotio/config.yaml`) |
| `--output, -o <fmt>` | Output format: `json` or `table` *(new)* |

### Authentication & setup

| Command | Description |
| --- | --- |
| `flotio login -e <email> -p <password>` | Authenticate; tokens stored in `~/.flotio/auth.json` |
| `flotio logout` | Log out and clear stored tokens |
| `flotio whoami` | Show the currently authenticated user |
| `flotio init [project-id]` | Mark current dir as a Flotio project (`.flotio.yaml`); interactive without an ID |
| `flotio configure --host <url>` | Set defaults in `~/.flotio/config.yaml` |
| `flotio version` | Print version and check for updates |
| `flotio update` | Self-update to the latest GitHub release |
| `flotio doctor` | Diagnose auth, config, and connectivity *(new)* |

### Projects — `flotio project`

```
flotio project list                # list all projects
flotio project get [id]            # show a project
flotio project create <name>       # create a project (accepts config flags below)
flotio project update [id]         # update name and/or config
flotio project delete [id]         # delete a project
```

`create` and `update` accept these configuration flags:

| Flag | Description |
| --- | --- |
| `--repo <url>` | Git repository URL |
| `--platform <p>` | Target platforms: `android`, `ios`, `web` (repeatable / comma-separated) |
| `--flutter-version <v>` | Flutter SDK version, e.g. `3.22.0` |
| `--build-mode <m>` | `debug`, `release`, or `profile` |
| `--project-path <path>` | Path to the Flutter project inside the repo |
| `--android-format <f>` | `apk` or `aab` |
| `--play-track <t>` | Google Play track: `internal`, `alpha`, `beta`, `production` |
| `--package-name <id>` | Android `applicationId` (e.g. `com.example.app`) |
| `--git-username <user>` | Git username for private repos |
| `--git-token <token>` | Git token/password (use `@file` to read from a file, e.g. `--git-token @./pat`) |

There is also a dedicated `flotio config` group (`get`, `update`, `delete`) for editing a project's configuration in place.

### Builds — `flotio build`

```
flotio build start [project-id]                 # trigger a build (--branch, --platform, --mode, --target)
flotio build list [project-id]                  # list builds for a project
flotio build logs   [project-id] <build-id>     # fetch build logs
flotio build cancel [project-id] <build-id>     # cancel a running build
flotio build download [project-id] <build-id>   # get a download URL for an artifact
flotio build delete [project-id] <build-id>     # delete a build and its artifacts
```

`start` flags: `--branch`, `--platform` (`android`/`ios`/`web`), `--mode` (`debug`/`release`/`profile`), `--target` (`apk`/`aab`).

### Releases — `flotio release` *(new)*

```
flotio release publish   # publish a build to Google Play
flotio release get       # show a release
flotio release list      # list releases
flotio release access    # manage release access
flotio release audit     # view the release audit log
```

### Build cache — `flotio cache` *(new)*

```
flotio cache purge     # purge the build cache
flotio cache metrics   # show cache metrics
flotio cache entries   # list cache entries
```

### Environment — `flotio env`

```
flotio env list                          # list env variables/files (--project to filter)
flotio env create <key> <value>          # create a var or file (--type env|file, --path)
flotio env update <id>                   # update (--value, --key, --type, --path, --base64)
flotio env delete <id>                   # delete an env asset
```

Values support the `@file` syntax to upload file contents (e.g. `flotio env create .env.prod @./.env.prod --type file --path .env`); file contents are base64-encoded automatically.

### Signing & publishing

```
flotio keystore list
flotio keystore create <name> --file <keystore.jks> --alias <alias> [--store-password p] [--key-password p]
flotio keystore delete <id>

flotio play list
flotio play create <name> --file <service-account.json>
flotio play delete <id>
```

### GitHub integration — `flotio github`

```
flotio github status                  # check GitHub installation status
flotio github connect <installation-id>
flotio github disconnect
flotio github repos                   # list accessible repositories
flotio github repo <owner/repo>       # show a repository file tree
```

### Flutter SDKs — `flotio flutter`

```
flotio flutter versions   # list available Flutter SDK versions on Flotio
```

---

## Configuration

The CLI stores everything under `~/.flotio/`:

| Path | Purpose | Permissions |
| --- | --- | --- |
| `~/.flotio/config.yaml` | Defaults (e.g. `host`) | `0644` |
| `~/.flotio/auth.json` | Access & refresh tokens | `0600` |

Per-project context is stored in **`.flotio.yaml`** at the project root (created by `flotio init`). When present, commands walk up the directory tree to find it and infer the project ID automatically.

The effective API host is resolved in this order:

1. `--host` flag
2. `host` in the config file
3. `FLOTIO_HOST` environment variable
4. built-in default (`api.flotio.ovh`)

You can also override the config file location with `--config` or the `FLOTIO_CONFIG` environment variable.

---

## Shell Completion

`flotio` is built with Cobra, which provides built-in completion generation. Generate and load completions for your shell:

**Bash:**

```sh
echo 'source <(flotio completion bash)' >> ~/.bashrc
# or, for a static file:
flotio completion bash > /etc/bash_completion.d/flotio   # system-wide (may need sudo)
```

**Zsh:**

```sh
echo 'source <(flotio completion zsh)' >> ~/.zshrc
```

**Fish:**

```sh
flotio completion fish > ~/.config/fish/completions/flotio.fish
```

**PowerShell:**

```powershell
flotio completion powershell | Out-String | Invoke-Expression
```

Run `flotio completion --help` for all options.

---

## Development

This project uses [devenv](https://devenv.sh/) (a nix-based dev shell). The `devenv.nix` file lives in the **parent** directory, so devenv-based commands run from there.

```sh
cd <path-to>/setup          # the directory containing devenv.nix and the cli/ folder
devenv shell                # enter the dev shell
```

Common tasks (run from `cli/` inside the dev shell):

| Command | Description |
| --- | --- |
| `make build` | Build `bin/flotio` with the current version (via devenv) |
| `make build-go` | Same, but using plain `go build` (no devenv required) |
| `make build-dev` | Build without ldflags (version reports `dev`) |
| `make test` | Run the test suite |
| `make run ARGS="version"` | Run the CLI from source |
| `make clean` | Remove build artifacts |
| `make install` | Build and install to `$GOBIN` |

> The canonical build (matching CI) is:
>
> ```sh
> cd <path-to>/setup && devenv shell go build -C cli \
>   -ldflags "-X github.com/flotio-dev/cli/cmd.version=<X.Y.Z>" -o bin/flotio .
> ```

### CI & releases

The [`.github/workflows/build.yml`](.github/workflows/build.yml) workflow runs on every push and PR. Pushing a `v*` tag cross-compiles binaries for `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, and `windows/amd64`, then publishes a GitHub Release with auto-generated notes. Releases power both the install scripts and `flotio update`.

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository and create a feature branch.
2. Make your changes, keeping commands consistent with the existing Cobra structure in [`cmd/`](cmd).
3. Add or update tests where applicable (`make test`).
4. Open a pull request describing the change.

For larger changes, please open an issue first to discuss the approach.

---

## License

License to be determined. See `LICENSE` once added.
