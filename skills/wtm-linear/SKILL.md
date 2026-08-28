---
name: wtm-linear
description: Fan a Linear board (or any list of Linear issues) out into parallel wtm worktrees, each running its own claude session seeded with that issue's task, branch-named the way Linear names branches. Use when the user pastes/references a Linear board, project, or set of issues and wants them worked on in parallel, or asks to "spin up worktrees for these tickets".
---

# wtm-linear: parallel Linear issue fan-out via wtm

`wtm` (see its README/USAGE) runs one `claude` process per git worktree,
each in its own detached tmux session, with a TUI that shows fleet-wide
status (Running / Needs input / Idle / Stopped) without attaching to every
session. This skill drives that mechanism headlessly to turn a batch of
Linear issues into a batch of parallel, independently-working agents.

This skill only **fans out**. It never merges, pushes, or closes issues —
those stay manual, reviewed actions via wtm's own `m` (merge) / `x` (remove)
keybindings in the TUI. Treat spinning up N autonomous coding agents as a
real action with real resource cost (N `claude` processes, N worktrees on
disk, API usage) — always show the plan and get explicit confirmation
before creating anything.

## Prerequisites

- `wtm` and `tmux` are installed and on `PATH`.
- The target repo has been opened with `wtm` at least once (it self-registers
  the current git repo into `~/.config/wtm/config.yaml` on first run), or you
  know its `--repo` name / are running from inside it.
- Ideally a Linear MCP integration is connected in this session. If not,
  ask the user to paste the issue list (identifier, title, description).

## Procedure

1. **Resolve the issue list.** If a Linear MCP integration is available, use
   it to fetch the board/project's issues: identifier (e.g. `ENG-123`),
   title, description/acceptance criteria, and `branchName` if the tool
   exposes it (Linear auto-generates one per issue). Otherwise, use what the
   user pasted directly — don't guess at issues that weren't given to you.

2. **Derive a branch name per issue.**
   - Prefer the issue's own `branchName` from Linear if you have it — that's
     the naming convention the user asked for.
   - Otherwise construct one yourself: lowercase the identifier, slugify the
     title (ASCII, hyphens, no punctuation), join with a hyphen —
     e.g. `ENG-123: Fix login redirect loop` → `eng-123-fix-login-redirect-loop`.
   - If two issues would collide on the same branch name, disambiguate
     (append a short suffix) rather than silently overwriting one.

3. **Show the plan and confirm before creating anything.** List, per issue:
   repo, branch name, base ref (default: let `wtm` auto-detect via
   `gitops.DefaultBaseRef` — don't pass `--base` unless the user specified
   one). Get explicit go-ahead from the user, especially for more than a
   handful of issues at once (each is a full `claude` process — mind local
   CPU/memory and API usage).

4. **Launch one worktree + session per issue:**

   ```sh
   wtm new <branch> --repo <repo-name> --launch --prompt "<seed prompt>"
   ```

   Seed prompt template (fill in the issue's real fields):

   ```
   You are working autonomously in an isolated git worktree on branch <branch>.
   Your task is Linear issue <identifier>: <title>.

   <description / acceptance criteria>

   Implement the change, run the project's tests, and commit your work with a
   descriptive message referencing <identifier>. Do not push or merge — a
   human will review and merge this branch later. If you hit a decision only
   a human can make, or you're blocked, stop and clearly state what you need
   instead of guessing.
   ```

   Run these sequentially, not concurrently as a batch of parallel tool
   calls — each is a `git worktree add` against the same repo, and git
   worktree metadata writes aren't safe to race. The `claude` sessions
   themselves *do* then run in parallel, in the background, once launched.

   If a `wtm new` call fails (e.g. branch/worktree already exists), report
   that issue's failure and continue with the rest rather than aborting the
   whole batch.

5. **Report back a table**: issue → branch → tmux session name (`wtm`
   derives it as `wtm-<repo>-<branch>`, sanitized). Tell the user these run
   independently of this conversation — closing it doesn't stop them.

6. **Point at how to monitor, don't babysit.** Don't attach to sessions or
   poll them in a loop yourself. Tell the user to run `wtm` (TUI, shows
   live 🟢/🟡/⚪/⚫ status per row and lets them jump to any row needing
   input with `a`) or `wtm ls` for a quick headless snapshot. If asked to
   check status yourself, a single `wtm ls` is fine; don't attach into a
   tmux session non-interactively to "peek" — that's what the TUI/`ls` are
   for.

## Notes

- `wtm new` also accepts `--base <ref>` to override the auto-detected base,
  and works with `--repo` omitted when run from inside the target repo.
- `--prompt` requires `--launch`; without `--launch`, `wtm new` just creates
  the worktree/branch and leaves it session-less (equivalent to the TUI's
  plain `n`, not `N`).
- Nothing here is Linear-specific at the `wtm` level — `wtm` itself has no
  knowledge of Linear. This skill is the only place that convention lives,
  by design (see the project's `wtm-plan.md`: multi-agent-type / issue
  tracker integrations are kept out of `wtm`'s Go core).
