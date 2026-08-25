# Using wtm

`wtm` is a terminal-first manager for running several coding-agent sessions
in parallel, one per git worktree, without leaving a single TUI. This is a
practical guide to its features; see [README.md](./README.md) for the design
rationale and [wtm-plan.md](./wtm-plan.md) for the full design doc.

## Install

From a checkout of this repo:

```sh
go install ./cmd/wtm
```

This drops a `wtm` binary into `$(go env GOBIN)` (usually `~/go/bin`) —
make sure that's on your `PATH`. `wtm` also shells out to `git` and `tmux`,
so both need to be installed.

Re-run `go install ./cmd/wtm` any time you pull new changes, to update the
installed binary.

## Getting started

`wtm` operates on whatever git repos it knows about, tracked in a small
config file at `$XDG_CONFIG_HOME/wtm/config.yaml` (defaults to
`~/.config/wtm/config.yaml`). Run it from inside any git repo and that repo
is registered automatically:

```sh
cd ~/code/some-project
wtm
```

New worktrees for a repo are created under `<worktree_root>/<repo-name>/<branch>`,
where `worktree_root` defaults to `~/wt` (also configurable in
`config.yaml`).

Run `wtm` again from a *different* repo and it registers that one too — a
single `wtm` instance manages every repo you've opened it from, all in one
fleet view.

## The fleet view

The sidebar lists every worktree across every registered repo, each tagged
with a live state icon derived from polling its tmux pane:

| Icon | State | Meaning |
|------|-------------|--------------------------------------------|
| 🟢 | Running | the pane's content is actively changing |
| 🟡 | Needs Input | the pane has gone quiet mid-task — likely waiting on you |
| ⚪ | Idle | pane content is stable, nothing pending |
| ⚫ | Stopped | no tmux session running for this worktree |

The fleet bar at the top surfaces a count of rows currently in **Needs
Input** — press `a` any time to jump straight to the next one.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `k`, ↓ / ↑ | move the cursor |
| `enter` | attach to the selected worktree's session |
| `tab` | cycle the session / diff / activity panes |
| `n` | new worktree |
| `N` | new worktree, and immediately launch `claude` in it |
| `m` | merge the worktree's branch into its base ref |
| `x` | remove the worktree |
| `e` | open the worktree's path in `$EDITOR` |
| `r` | refresh git status |
| `s` | (diff tab) toggle split / unified diff view |
| `[` / `]` | (diff tab) previous / next file |
| `a` | jump to the next row needing input |
| `/` | filter the list |
| `:` | command palette |
| `?` | keybinds & settings overlay |
| `q` | quit (sessions keep running in the background) |

## Creating a worktree

Press `n` (or `N` to also launch `claude` right away). You get two fields:

- **branch** — the new branch name (e.g. `feat/my-branch`).
- **base ref** — what to branch from. `tab` switches between the two
  fields; leave base ref blank to use the repo's default (its upstream if
  the current branch tracks one, else `origin/HEAD`, else a local `main` or
  `master`).

`enter` creates the worktree with `git worktree add -b <branch> <path>
<base-ref>`. With `N`, a tmux session running `claude` starts immediately
in it; with plain `n`, the worktree exists but has no session until you
first attach.

## Attaching, and the embedded terminal

Press `enter` on a worktree to attach. If no tmux session exists yet for it,
one is created running `claude`. Unlike a plain `tmux attach`, `wtm` doesn't
suspend itself and hand off your terminal — it embeds a live PTY-backed tmux
client directly in its own pane (see `internal/termpty`), so the rest of the
TUI frame (header, sidebar, tabs) stays visible while you interact with the
session.

While attached:

- Every keystroke goes to the session, exactly as if you'd run
  `tmux attach` yourself.
- **`Ctrl-b d`** (tmux's own detach prefix) or **pressing `Esc` twice
  quickly** detaches wtm's view. The session itself, and everything running
  in it, keeps going in the background. Only the *first* `Esc` is forwarded
  to the session; the second one (the one that actually triggers the
  detach) is swallowed by `wtm` rather than sent on — so double-`Esc` won't
  also trigger whatever a program inside the session (e.g. `claude`) does
  on its own double-`Esc`.
- Because it's a real tmux client underneath, **tmux's own commands work
  for running multiple terminals in the same worktree**:
  - `Ctrl-b c` — open a new window running a plain shell (not `claude` —
    only the session's first window runs the launch command).
  - `Ctrl-b n` / `Ctrl-b p` / `Ctrl-b 0`–`9` — switch between windows.
  - `Ctrl-b %` / `Ctrl-b "` — split the current window into side-by-side /
    stacked panes.

  (These use tmux's default prefix, `Ctrl-b`; if you've customized
  `~/.tmux.conf` to remap it, your own prefix applies here too.)

- Resizing your terminal reflows the embedded pane and the PTY together, so
  whatever's running inside (including `claude`) sees the new size.

## Reviewing and merging

`tab` cycles the main pane between three views for the selected worktree:

- **session** — the live attached terminal (see above), or a read-only tail
  of the pane's last output when not attached.
- **diff** — the worktree's changes against its base ref
  (`git diff base...HEAD`). A file list on the left, the selected file's
  diff on the right; `]` / `[` move between files. The right pane defaults
  to a unified diff; press `s` (or click the hint in the footer) to switch
  to a side-by-side split view like GitHub's or VS Code's — old lines on
  the left, new lines on the right, paired up line-by-line. `s` again (or
  the same click) switches back.
- **activity** — an in-memory feed of fleet events for this session
  (created, state transitions, merged, removed). Not persisted — it only
  covers the current `wtm` run.

Press `m` to merge a worktree's branch into its base ref: this runs
`git merge --no-ff <branch>` **on the base repo's own checkout**, so make
sure that checkout currently has the base ref checked out. The worktree and
its session are left untouched — merging doesn't remove either.

Press `x` to remove a worktree (`git worktree remove --force`, plus killing
its tmux session if one exists). Both `m` and `x` ask for confirmation
(`y`/`enter` to proceed, `n`/`esc` to cancel) before doing anything.

## Filtering and the command palette

- `/` opens a live filter over the sidebar — it matches against branch,
  repo name, and path as you type; `enter` or `esc` closes the filter
  (closing doesn't clear it — clear the input to see everything again).
- `:` opens a command palette listing every action `wtm` supports, with its
  keybind, filterable by typing; `↑`/`↓` to move, `enter` to run.

## Headless mode

For scripting, or a quick check without opening the TUI:

```sh
wtm ls
```

Prints one line per worktree across every registered repo (state icon, repo
name, branch, path) and exits — no tmux attach, no interactivity.

## Design notes

- **No database.** All state is derived live from `git worktree list
  --porcelain` and `tmux ls` / `tmux capture-pane` on every poll — quit
  `wtm` and nothing is lost, because nothing besides the config file was
  ever wtm's to lose.
- **Sessions outlive the TUI.** Closing `wtm` (`q`) never touches tmux
  sessions; they keep running until you kill them (`x` on their worktree,
  or `tmux kill-session` yourself).
