package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/config"
	"github.com/Mlcarvalho1/wtm/internal/gitops"
	"github.com/Mlcarvalho1/wtm/internal/session"
	"github.com/Mlcarvalho1/wtm/internal/watch"
)

type mode int

const (
	modeList mode = iota
	modeNewBranch
	modeNewBase
	modeConfirmRemove
	modeFilter
)

type Model struct {
	cfg config.Config

	table    table.Model
	worktree []Worktree
	mode     mode

	branchInput textinput.Model
	baseInput   textinput.Model
	newRepo     config.Repo
	autoLaunch  bool

	removeTarget Worktree

	filterInput textinput.Model
	filterQuery string

	paneSnapshot watch.Snapshot

	width       int
	height      int
	showPreview bool

	status string
	err    error
	busy   bool
}

func New(cfg config.Config) Model {
	bi := textinput.New()
	bi.Placeholder = "feat/my-branch"
	bi.CharLimit = 200

	ri := textinput.New()
	ri.Placeholder = "(default: current HEAD)"
	ri.CharLimit = 200

	fi := textinput.New()
	fi.Placeholder = "branch/repo/path substring"
	fi.CharLimit = 200

	return Model{
		cfg:         cfg,
		table:       newTable(),
		branchInput: bi,
		baseInput:   ri,
		filterInput: fi,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadWorktreesCmd(m.cfg), watch.Tick())
}

// --- messages ---

type worktreesLoadedMsg struct {
	rows []Worktree
	errs []error
}

type actionDoneMsg struct{ status string }
type actionErrMsg struct{ err error }
type attachFinishedMsg struct{ err error }
type editorFinishedMsg struct{ err error }

type pollResultMsg struct {
	updates  []watch.Update
	snapshot watch.Snapshot
}

// --- commands ---

func loadWorktreesCmd(cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		rows, errs := LoadWorktrees(cfg)
		return worktreesLoadedMsg{rows: rows, errs: errs}
	}
}

func addWorktreeCmd(cfg config.Config, repo config.Repo, branch, baseRef string, autoLaunch bool) tea.Cmd {
	return func() tea.Msg {
		path := filepath.Join(cfg.WorktreeRoot, repo.Name, branch)
		if err := gitops.AddWorktree(repo.Path, path, branch, baseRef, true); err != nil {
			return actionErrMsg{err}
		}
		if !autoLaunch {
			return actionDoneMsg{status: fmt.Sprintf("created worktree %s", branch)}
		}
		name := SessionName(repo.Name, branch)
		if err := session.New(name, path, "claude"); err != nil {
			return actionErrMsg{err}
		}
		return actionDoneMsg{status: fmt.Sprintf("created worktree %s and launched claude", branch)}
	}
}

func removeWorktreeCmd(wt Worktree) tea.Cmd {
	return func() tea.Msg {
		if err := session.Kill(wt.Session); err != nil {
			return actionErrMsg{err}
		}
		if err := gitops.RemoveWorktree(wt.RepoPath, wt.Path, true); err != nil {
			return actionErrMsg{err}
		}
		_ = gitops.DeleteBranch(wt.RepoPath, wt.Branch, true)
		return actionDoneMsg{status: fmt.Sprintf("removed worktree %s", wt.Branch)}
	}
}

// pollCmd runs watch.Poll inside a tea.Cmd (bubbletea's own goroutine) and
// reports the result back as a message — the model itself never touches
// tmux directly on a timer, keeping the render loop unblocked.
func pollCmd(names []string, prev watch.Snapshot) tea.Cmd {
	return func() tea.Msg {
		updates, next := watch.Poll(names, prev)
		return pollResultMsg{updates: updates, snapshot: next}
	}
}

// liveSessionNames returns the session names of rows we already believe have
// a running session, so the ticker doesn't spend a tmux call every 2s on
// worktrees that have never had a session launched.
func liveSessionNames(wts []Worktree) []string {
	var names []string
	for _, w := range wts {
		if w.State != watch.StateStopped {
			names = append(names, w.Session)
		}
	}
	return names
}

func applyPollUpdates(wts []Worktree, updates []watch.Update) {
	byName := make(map[string]watch.Update, len(updates))
	for _, u := range updates {
		byName[u.Name] = u
	}
	for i := range wts {
		if u, ok := byName[wts[i].Session]; ok {
			wts[i].State = u.State
			if u.Pane != "" {
				wts[i].LastPane = u.Pane
			}
		}
	}
}

// Update handles all key/message routing. Attach is the one path that
// suspends the TUI via tea.ExecProcess to hand the terminal to `tmux attach`.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 6)
		m.table.SetWidth(m.tableWidth())
		return m, nil

	case worktreesLoadedMsg:
		m.worktree = msg.rows
		m.table.SetRows(rowsFromWorktrees(m.visibleWorktrees()))
		m.busy = false
		if len(msg.errs) > 0 {
			m.err = msg.errs[0]
		} else {
			m.err = nil
		}
		return m, nil

	case actionDoneMsg:
		m.busy = false
		m.status = msg.status
		m.err = nil
		return m, loadWorktreesCmd(m.cfg)

	case actionErrMsg:
		m.busy = false
		m.err = msg.err
		return m, nil

	case attachFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, loadWorktreesCmd(m.cfg)

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case watch.TickMsg:
		names := liveSessionNames(m.worktree)
		return m, tea.Batch(pollCmd(names, m.paneSnapshot), watch.Tick())

	case pollResultMsg:
		applyPollUpdates(m.worktree, msg.updates)
		m.paneSnapshot = msg.snapshot
		m.table.SetRows(rowsFromWorktrees(m.visibleWorktrees()))
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeNewBranch:
		return m.handleNewBranchKey(msg)
	case modeNewBase:
		return m.handleNewBaseKey(msg)
	case modeConfirmRemove:
		return m.handleConfirmRemoveKey(msg)
	case modeFilter:
		return m.handleFilterKey(msg)
	default:
		return m.handleListKey(msg)
	}
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Refresh):
		m.busy = true
		m.status = ""
		return m, loadWorktreesCmd(m.cfg)

	case key.Matches(msg, keys.New):
		return m.startNewWorktree(false)

	case key.Matches(msg, keys.NewAttach):
		return m.startNewWorktree(true)

	case key.Matches(msg, keys.Remove):
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		m.removeTarget = wt
		m.mode = modeConfirmRemove
		return m, nil

	case key.Matches(msg, keys.Attach):
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.attach(wt)

	case key.Matches(msg, keys.Preview):
		m.showPreview = !m.showPreview
		if m.width > 0 {
			m.table.SetWidth(m.tableWidth())
		}
		return m, nil

	case key.Matches(msg, keys.NextNeedsInput):
		m.jumpToNextNeedsInput()
		return m, nil

	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filterInput.SetValue(m.filterQuery)
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
		return m, nil

	case key.Matches(msg, keys.Edit):
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.openEditor(wt)
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// startNewWorktree begins the n/N prompt flow. autoLaunch remembers whether
// N (vs plain n) was pressed, so the base-ref step knows to launch `claude`
// in the worktree immediately after creating it.
func (m Model) startNewWorktree(autoLaunch bool) (tea.Model, tea.Cmd) {
	if len(m.cfg.Repos) == 0 {
		m.err = fmt.Errorf("no repos configured")
		return m, nil
	}
	m.newRepo = m.repoForNewWorktree()
	m.autoLaunch = autoLaunch
	m.mode = modeNewBranch
	m.branchInput.SetValue("")
	m.branchInput.Focus()
	m.status = ""
	m.err = nil
	return m, nil
}

// repoForNewWorktree defaults to the currently highlighted row's repo,
// falling back to the first registered repo.
func (m Model) repoForNewWorktree() config.Repo {
	if wt, ok := m.selectedWorktree(); ok {
		for _, r := range m.cfg.Repos {
			if r.Path == wt.RepoPath {
				return r
			}
		}
	}
	return m.cfg.Repos[0]
}

// visibleWorktrees is m.worktree narrowed by the active filter, if any —
// this is what the table actually displays, so cursor positions and jumps
// must index into this, not the full unfiltered set.
func (m Model) visibleWorktrees() []Worktree {
	if m.filterQuery == "" {
		return m.worktree
	}
	q := strings.ToLower(m.filterQuery)
	var out []Worktree
	for _, w := range m.worktree {
		haystack := strings.ToLower(w.RepoName + " " + w.Branch + " " + w.Path)
		if strings.Contains(haystack, q) {
			out = append(out, w)
		}
	}
	return out
}

func (m Model) selectedWorktree() (Worktree, bool) {
	rows := m.visibleWorktrees()
	i := m.table.Cursor()
	if i < 0 || i >= len(rows) {
		return Worktree{}, false
	}
	return rows[i], true
}

// tableWidth returns how wide the table itself should be: the full terminal
// width normally, or a narrower left column when the preview pane is shown.
func (m Model) tableWidth() int {
	if m.width == 0 {
		return 0
	}
	if !m.showPreview {
		return m.width
	}
	return max(m.width*6/10, 40)
}

// jumpToNextNeedsInput moves the cursor to the next row (wrapping around)
// whose session is waiting on a human, so a user can cycle through everything
// that needs attention without scanning the whole fleet by eye.
func (m *Model) jumpToNextNeedsInput() {
	rows := m.visibleWorktrees()
	n := len(rows)
	if n == 0 {
		return
	}
	start := m.table.Cursor()
	for i := 1; i <= n; i++ {
		idx := (start + i) % n
		if rows[idx].State == watch.StateNeedsInput {
			m.table.SetCursor(idx)
			return
		}
	}
}

func (m Model) handleNewBranchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "enter":
		if m.branchInput.Value() == "" {
			return m, nil
		}
		m.mode = modeNewBase
		m.baseInput.SetValue("")
		m.baseInput.Focus()
		return m, nil
	}
	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

func (m Model) handleNewBaseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "enter":
		m.mode = modeList
		m.busy = true
		branch := m.branchInput.Value()
		baseRef := m.baseInput.Value()
		return m, addWorktreeCmd(m.cfg, m.newRepo, branch, baseRef, m.autoLaunch)
	}
	var cmd tea.Cmd
	m.baseInput, cmd = m.baseInput.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmRemoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeList
		m.busy = true
		return m, removeWorktreeCmd(m.removeTarget)
	default:
		m.mode = modeList
		return m, nil
	}
}

// handleFilterKey applies the filter live as the user types, so the table
// narrows immediately rather than waiting for a confirming Enter.
func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.filterQuery = ""
		m.filterInput.SetValue("")
		m.table.SetRows(rowsFromWorktrees(m.visibleWorktrees()))
		return m, nil
	case "enter":
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterQuery = m.filterInput.Value()
	m.table.SetRows(rowsFromWorktrees(m.visibleWorktrees()))
	return m, cmd
}

// attach ensures a tmux session exists for wt (creating one running `claude`
// if needed), then suspends the TUI and execs `tmux attach-session`,
// resuming this model once the user detaches (Ctrl-b d) or the session ends.
func (m Model) attach(wt Worktree) (tea.Model, tea.Cmd) {
	if !session.Exists(wt.Session) {
		if err := session.New(wt.Session, wt.Path, "claude"); err != nil {
			m.err = err
			return m, nil
		}
	}
	cmd := exec.Command("tmux", "attach-session", "-t", wt.Session)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return attachFinishedMsg{err: err}
	})
}

// openEditor suspends the TUI to open wt's path in $EDITOR (falling back to
// vi), resuming this model once the editor exits.
func (m Model) openEditor(wt Worktree) (tea.Model, tea.Cmd) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, wt.Path)
	cmd.Dir = wt.Path
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// body renders the table alone, or the table with the preview pane for the
// highlighted row to its right when toggled on with tab.
func (m Model) body() string {
	if !m.showPreview {
		return m.table.View()
	}
	wt, ok := m.selectedWorktree()
	if !ok {
		return m.table.View()
	}
	previewWidth := m.width - m.tableWidth() - 2
	height := m.table.Height() + 2
	return lipgloss.JoinHorizontal(lipgloss.Top, m.table.View(), "  ", renderPreview(wt, previewWidth, height))
}

func (m Model) View() string {
	var b string
	b += styleTitle.Render("wtm") + "  " + styleHelp.Render("worktree/agent manager") + "\n\n"
	b += m.body() + "\n\n"

	switch m.mode {
	case modeNewBranch:
		b += stylePrompt.Render("New worktree — branch name:") + "\n" + m.branchInput.View() + "\n"
	case modeNewBase:
		b += stylePrompt.Render("Base ref for "+m.branchInput.Value()+":") + "\n" + m.baseInput.View() + "\n"
	case modeConfirmRemove:
		b += stylePrompt.Render(fmt.Sprintf("Remove worktree %q at %s? (kills session, deletes branch) [y/N]",
			m.removeTarget.Branch, m.removeTarget.Path)) + "\n"
	case modeFilter:
		b += stylePrompt.Render("Filter:") + "\n" + m.filterInput.View() + "\n"
	default:
		if m.err != nil {
			b += styleError.Render("error: "+m.err.Error()) + "\n"
		} else if m.status != "" {
			b += styleHelp.Render(m.status) + "\n"
		} else if m.filterQuery != "" {
			b += styleHelp.Render(fmt.Sprintf("filtered: %q (esc to clear)", m.filterQuery)) + "\n"
		}
		b += styleHelp.Render("n new  ·  N new+launch  ·  enter attach  ·  x remove  ·  tab preview  ·  a next needs-input  ·  / filter  ·  e edit  ·  r refresh  ·  q quit") + "\n"
	}

	return b
}
