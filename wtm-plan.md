# wtm — Lightweight Worktree/Agent Manager (Go)

A lightweight, terminal-first alternative to Orca. Orca is an Electron "ADE" that
runs coding agents in isolated git worktrees with embedded terminals, diffs,
browser, GitHub/Linear panels, etc. — that's exactly why it eats memory. This
project cuts everything down to the actual need: **run several Claude Code
sessions in parallel, one per git worktree, on the local machine, with fleet-wide
visibility, from a fast Bubble Tea TUI.**

## Scope (decided)

- **Interaction model:** Terminal UI (lazygit/k9s style), `wtm` with no args opens it.
- **Agent:** Claude Code only (no multi-agent-type abstraction needed yet).
- **Locality:** Local worktrees only — no SSH/remote for v1.
- **"Multiple agents" means:** several `claude` processes running in parallel,
  each in its own worktree/tmux session, with the TUI giving fleet-wide status
  (running / needs input / idle / stopped) without having to attach to each one.

## Core design principle

**Don't reimplement a terminal emulator. Let `tmux` own persistence; let Go own
orchestration.**

- Each worktree gets a detached `tmux` session running `claude` inside it.
- The TUI lists worktrees + their tmux sessions; `enter` attaches by suspending
  the TUI process and exec'ing `tmux attach -t <session>` (Bubble Tea's
  `tea.ExecProcess`), resuming the TUI on detach (`Ctrl-b d`).
- Sessions live independently of the TUI — close `wtm`, agents keep running.
- The Go process never holds a PTY buffer or parses ANSI streams for rendering
  a terminal — it only ever calls `tmux capture-pane -p` to peek at plain text.

This is what keeps memory near-zero: no Chromium, no embedded terminal
emulation, no persistent per-pane buffers — just process orchestration plus
cheap polling of tmux's own state.

No database. State is always derived live from:
- `git worktree list --porcelain` (+ status per worktree)
- `tmux ls` / `tmux capture-pane`

Nothing to get out of sync, nothing to migrate.

## Stack

| Concern | Choice | Why |
|---|---|---|
| TUI framework | `charmbracelet/bubbletea` + `bubbles` + `lipgloss` | Elm-architecture, mature, used by Soft Serve/Glow/Crush, small footprint |
| Git ops | shell out to real `git` via `os/exec` | `go-git` has poor/no worktree support; real git is the source of truth anyway |
| Session mux | shell out to `tmux` | zero-cost persistence/attach-detach vs. writing a PTY manager |
| Config | `gopkg.in/yaml.v3` | trivial `~/.config/wtm/config.yaml` |
| CLI entry (optional) | `spf13/cobra` | `wtm new`, `wtm ls` work headless too; bare `wtm` launches the TUI |

## Data model

```go
type Repo struct {
    Name string
    Path string // main checkout
}

type SessionState int

const (
    StateStopped SessionState = iota
    StateIdle
    StateRunning
    StateNeedsInput
)

type Worktree struct {
    Repo   string
    Branch string
    Path   string // e.g. ~/wt/amigo-app/feat-tap-to-pay

    Dirty  bool
    Ahead  int
    Behind int

    Session  string       // tmux session name
    State    SessionState  // derived by watch/poll.go
    LastPane string        // last capture-pane snapshot (diff + preview source)
}
```

## Package layout

```
wtm/
  cmd/wtm/main.go
  internal/
    gitops/
      worktree.go   // add / remove / list, base-ref selection
      status.go     // dirty/ahead/behind parsing
    session/
      tmux.go       // create / attach / kill / list sessions
    watch/
      poll.go       // tea.Tick loop, capture-pane, diff vs LastPane, classify state
      patterns.go   // "needs input" detection (permission prompts, etc.)
    config/
      config.go     // load/save yaml, worktree root dir, repo registry
    tui/
      model.go      // root bubbletea model + Update/View
      list.go       // worktree table (bubbles/table)
      preview.go     // right-hand pane: tail of highlighted session's capture-pane
      keys.go        // keymap
      styles.go      // lipgloss styles
```

## Keybindings

- `j` / `k` — navigate rows
- `n` — new worktree (prompt: repo, branch name, base ref)
- `N` — new worktree **and** immediately launch `claude` in it (fast path for
  spinning up several parallel tasks)
- `enter` — attach (suspend TUI → `tmux attach`)
- `x` — remove worktree (kill session, `git worktree remove`, optional branch delete)
- `e` — open worktree path in `$EDITOR`
- `r` — refresh git status manually
- `tab` — toggle preview pane
- `a` — jump to next row in `NeedsInput` state
- `/` — filter list
- `q` — quit (sessions keep running in the background)

## Fleet status (the "multiple agents" feature)

Since several `claude` sessions run unattended in parallel, the TUI needs to
surface which ones need attention without manually attaching to each:

- Every ~2s, `watch/poll.go` runs `tmux capture-pane -p -t <session>` for each
  known session (only sessions that currently exist — no polling for stopped
  ones).
- Diff the new capture against `LastPane`:
  - unchanged + matches a "waiting" pattern (`? for shortcuts`,
    `Do you want to proceed`, permission prompts, etc.) → **NeedsInput**
  - unchanged, no waiting pattern → **Idle**
  - changed since last poll → **Running**
  - `tmux` reports no such session → **Stopped**
- Emits a `sessionUpdatedMsg{name, state, pane}` into Bubble Tea's own
  `Update` loop via `tea.Tick`, so it stays fully event-driven — no goroutine
  racing the renderer.
- Cost per tick: one non-blocking `exec` + a string diff per live session —
  negligible compared to holding a rendered PTY per pane (Orca's approach).

Row indicators:
- 🟢 Running 🟡 Needs input ⚪ Idle ⚫ Stopped

Preview pane (`tab`): shows the last N lines of `LastPane` for the highlighted
row — plain text tail, not a terminal emulation.

## What's deliberately cut vs Orca

- No embedded browser, no GitHub/Linear/PR panels, no diff review UI, no
  mobile companion, no SSH/remote worktrees, no multi-agent-type abstraction
  (Codex, etc.) — none of these are in current scope.
- No background polling when idle/unfocused beyond the fleet-status ticker
  above; git status itself only refreshes on keypress/focus, not on a timer.
- These can be added later without an architecture change — e.g. Linear
  ticket → branch-name templating is just a config/format detail, not a new
  subsystem, since you already have a Linear MCP integration to draw ticket
  IDs from.

## Phased build plan

1. **`gitops`** — worktree create/list/remove + status parsing. Unit-testable,
   no TUI dependency yet. Verify against a real repo from the CLI before
   touching Bubble Tea.
2. **`session`** — tmux create/attach/kill/list wrappers. Confirm
   `tea.ExecProcess` suspend/resume works cleanly for attach/detach.
3. **Bare-bones TUI** — `bubbles/table` list wired to `gitops` + `session`:
   `n`, `enter`, `x`, `q` working end to end. This is already a usable v0.
4. **`watch`** — fleet status ticker, row indicators, `NeedsInput` detection
   tuned against real Claude Code prompt text.
5. **Preview pane** — `tab` toggle, capture-pane tail rendering.
6. **Polish** — `N` (new + auto-launch), `a` (jump to needs-input), `/` filter,
   confirm dialogs on destructive actions, `$EDITOR` integration.
7. *(Later, optional)* Linear ticket → branch name templating; SSH targets if
   remote worktrees become a real need.

## Notes for implementation in Claude Code

- Build and test `gitops` and `session` as standalone packages with real
  `git`/`tmux` calls first (integration-style tests against a scratch repo) —
  the TUI should be the last layer added, not the first.
- Keep `watch`'s "needs input" pattern list in its own file (`patterns.go`)
  since it will need tuning against actual Claude Code CLI output over time.
- No persistent state file for v1 — everything derives from `git worktree
  list --porcelain` and `tmux ls` on demand, which avoids an entire class of
  sync bugs.
