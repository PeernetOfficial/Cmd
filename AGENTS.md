# AGENTS.md

Rules for AI agents working in this repository.

## What this repo is

`github.com/PeernetOfficial/Cmd` — the Peernet reference client, a command line program built on the core library (`github.com/PeernetOfficial/core`). It proves the core works and shows how to use it. Everything protocol-related lives in core; this repository only wraps it: command line parsing, the interactive console, the optional local web API (`/console`, `/shutdown`), file transfer helpers and the debug build. Public on GitHub; `origin` is the source of truth.

## Hard rules

1. **Diff first, commit on approval.** Show the owner the full `git diff` of every change and wait for an explicit approval before committing. No exceptions, including one-line fixes and edits he asked for in chat.
2. **One logical change per commit.** Message = what + why. Push immediately after committing.
3. **Master is live.** Every commit on master builds, passes `go vet` and `go test`, and has been run. Nothing half-done, no stubs, no `TODO` that papers over a problem, no silenced errors. Report what real execution returned, never what it should return.
4. **Protocol logic belongs in core, not here.** If a change needs new network, DHT, blockchain, transfer or search behaviour, it is a core change first; this repo only calls the exported API. No copies of core code.
5. **Public repo.** No credentials, hostnames, server names, infrastructure details or internal roadmap. Example addresses in docs are loopback or documentation addresses.
6. **No third-party contributions.** Pull requests from outside are not accepted and not merged; the history stays single-author.
7. **Simplicity wins.** This is a thin client. If a change grows a feature nobody asked for, cut it.
8. **Underspecified or wrong is reported, not worked around.** Say so plainly, document it, solve it with the owner.

## Layout

| Path | Purpose |
|---|---|
| `Main.go` | Entry point: flags (`-webapi`, `-apikey`, `-watchpid`), config struct, `core.Init`, startup. Config file name is hard-coded to `Config.yaml`. |
| `Command Line.go` | Interactive console: `help`, `status`, `peer list`, `search file`, `dht get/store`, `warehouse get/store`, `transfer list`, … |
| `Command Debug.go` | Debug console commands (connect, watch, bucket refresh, probe transfer). |
| `API.go` | Local web API on top of core's `webapi`: `/console` websocket and `/shutdown`. |
| `File Transfer.go`, `Block Transfer.go` | Console-side file and block transfer with progress output. |
| `Debug.go`, `Debug empty.go` | `/debug` pprof endpoints, compiled only with `-tags debug`. |
| `README.md` | Public documentation: build, use, config, web API, exit codes, firewall notes. |
| `API.md` | The `/console` endpoint. |

Exit codes are the `Exit*` constants from core; the table in README is kept in sync with them.

## Code conventions

- Go file names are capitalized words separated by spaces (`Command Line.go`, `File Transfer.go`); match that for new files.
- Every Go file starts with the header block:
  ```
  /*
  File Name:  <name>.go
  Copyright:  <year> Kleissner Ventures Ltd
  Author:     Peter Kleissner
  */
  ```
  Old `Peernet Foundation s.r.o.` / `Peernet s.r.o.` headers are updated when the file is touched for real work, not in a bulk sweep.
- Line endings: commit files as they are. Never convert an existing file's line endings, never add a `.gitattributes`; a line-ending-only change is noise in the diff and is rejected. `gofmt -l` flags CRLF files — if `gofmt -l` also lists the HEAD version of the file, it is only the line endings, not a formatting error.
- Console output is plain text, one fact per line, readable by a human and greppable; errors go to the output with the exit code from the table, never a bare panic.
- New dependencies need the owner's approval; keep the module small.

## Core version

`go.mod` pins a specific core commit. Bumping it is its own commit (`go get github.com/PeernetOfficial/core@master && go mod tidy`, message states what the new core brings) and is done after the core change is on core's master, never before. For local development against uncommitted core changes use a `replace github.com/PeernetOfficial/core => ../core` directive in the working tree and never commit it.

## Build & verify

```bash
go build ./...
go vet ./...
go test ./...
go build -tags debug ./...
```

All must be clean before a diff is shown. A changed command is then run for real: start the built binary, execute the command, and report the actual output. Changes that touch peer interaction are verified against a second running peer. `Config.yaml`, `data/` and binaries from test runs stay out of the repo (`.gitignore` covers `*.exe`, `*debug_bin`, `*.yaml`, `data/`, `.vscode`).

Release builds: `go build -trimpath -ldflags "-s"` (see README for the Windows headless variant and code signing).

## Git workflow

- Remote: `https://github.com/PeernetOfficial/Cmd.git`, branch `master`. Wrap git network calls as `GIT_TERMINAL_PROMPT=0 timeout 60 git …`.
- `git pull --ff-only` at session start and again right before every commit; `git status --short` before editing — an unexpectedly dirty tree is reported, never stashed over.
- **Large changes go through an issue and a pull request.** A change that cannot land as one reviewable commit on master — multi-commit work, anything touching the wire format, cryptography or on-disk compatibility, or a diff too big to review in chat — is done on a branch and merged via a pull request. Before the branch is created, an issue describes what changes, why, and the scope; the pull request references it (`Closes #N`). Commits on the branch follow the same rules as master (one logical change each, builds, `go vet` and `go test` clean). Merge with a merge commit, never squash, so the step-by-step history is kept. The merge needs the owner's approval like any commit. Small changes stay as they are: diff in chat, commit on approval, straight to master.
- There is no own version number: the user agent is `appName + "/" + core.Version`, so the version shown is the pinned core's. Bumping the core pin is the version bump.
