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

## Build

```sh
go build ./...
```

## Keybindings

- `j` / `k` — navigate rows
- `n` — new worktree
- `N` — new worktree and immediately launch `claude` in it
- `enter` — attach (suspend TUI → `tmux attach`)
- `x` — remove worktree
- `e` — open worktree path in `$EDITOR`
- `r` — refresh git status
- `tab` — toggle preview pane
- `a` — jump to next row needing input
- `/` — filter list
- `q` — quit (sessions keep running in the background)
