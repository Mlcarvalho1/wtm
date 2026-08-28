# wtm — Lightweight Worktree/Agent Manager

A lightweight, terminal-first alternative to Orca. Run several [Claude Code](https://github.com/anthropics/claude-code)
sessions in parallel, one per git worktree, on your local machine, with
fleet-wide visibility, from a fast [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI.

## Why

Orca is an Electron "ADE" that runs coding agents in isolated git worktrees
with embedded terminals, diffs, browser, GitHub/Linear panels, etc. — that's
exactly why it eats memory. `wtm` cuts everything down to the actual need.

## Design

- Each worktree gets a detached `tmux` session running `claude` inside it.
- The TUI lists worktrees + their tmux sessions; `enter` attaches by
  suspending the TUI and exec'ing `tmux attach -t <session>`, resuming the
  TUI on detach.
- Sessions live independently of the TUI — close `wtm`, agents keep running.
- No database: state is always derived live from `git worktree list
  --porcelain` and `tmux ls`/`tmux capture-pane`.

See [wtm-plan.md](./wtm-plan.md) for the full design doc.

## Stack

- `charmbracelet/bubbletea` + `bubbles` + `lipgloss` — TUI
- Shells out to real `git` and `tmux` via `os/exec`
- `spf13/cobra` — optional headless CLI commands
- `gopkg.in/yaml.v3` — config

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Mlcarvalho1/wtm/main/install.sh | bash
```

An interactive wizard for macOS and Linux: checks for `git`/`tmux`/`go`,
offers to install anything missing, builds and installs `wtm`, and sets up
`PATH` and a default config. See [USAGE.md](./USAGE.md#install) for
details and a manual-install alternative.

## Build

```sh
go build ./...
```

## Versioning & updates

`wtm` follows semver, currently `0.0.1`. Check the installed version and
look for a newer one with:

```sh
wtm version   # print the installed version
wtm update    # check GitHub for a newer release and reinstall it
```

`wtm update` checks the repo's latest GitHub release (falling back to its
newest tag), and if it's newer than what's installed, reinstalls via
`go install github.com/Mlcarvalho1/wtm/cmd/wtm@v<version>` — the same
mechanism [install.sh](./install.sh) uses. It requires a Go toolchain on
`PATH`.

## Keybindings

- `j` / `k` — navigate rows
- `n` — new worktree
- `N` — new worktree and immediately launch `claude` in it
- `enter` — attach (suspend TUI → `tmux attach`)
- `tab` — cycle the session / diff / activity panes
- `m` — merge the selected worktree's branch into its base ref
- `x` — remove worktree
- `e` — open worktree path in `$EDITOR`
- `r` — refresh git status
- `a` — jump to next row needing input
- `/` — filter list
- `:` — command palette
- `?` — keybinds & settings
- `q` — quit (sessions keep running in the background)

## Panes

- **session** — tail of the selected worktree's tmux pane, plus its path,
  base ref, and git status.
- **diff** — the selected worktree's changes against its base ref
  (`git diff base...HEAD`), file list on the left, unified diff on the
  right (`]` / `[` to switch files); `m` merges, `x` discards.
- **activity** — an in-memory, session-lifetime feed of fleet events
  (worktrees created, state transitions, merges, removals). Not persisted —
  consistent with wtm's "no database" design.
