package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/activity"
	"github.com/Mlcarvalho1/wtm/internal/config"
	"github.com/Mlcarvalho1/wtm/internal/gitops"
	"github.com/Mlcarvalho1/wtm/internal/session"
	"github.com/Mlcarvalho1/wtm/internal/termpty"
	"github.com/Mlcarvalho1/wtm/internal/watch"
)

type mode int

const (
	modeList mode = iota
	modeFilter
	modeNewWorktree
	modeConfirm
	modePalette
	modeHelp
	modeAttached
)

// doubleEscWindow is how quickly two Escape presses must follow each other
// while attached to count as the local "detach" gesture rather than two
// ordinary Escape keystrokes forwarded to the session.
const doubleEscWindow = 500 * time.Millisecond

// doubleClickWindow mirrors doubleEscWindow for mouse clicks: two left
// clicks on the same sidebar row within this long count as a double-click
// (attach), matching the design's onDoubleClick handler.
const doubleClickWindow = 500 * time.Millisecond

type tabKind int

const (
	tabSession tabKind = iota
	tabDiff
	tabActivity
)

func nextTab(t tabKind) tabKind {
	switch t {
	case tabSession:
		return tabDiff
	case tabDiff:
		return tabActivity
	default:
		return tabSession
	}
}

type confirmKind int

const (
	confirmRemove confirmKind = iota
	confirmMerge
)

// commandPalette is the ':' palette's command list — label, keybind hint.
// Only actions the app actually implements belong here.
var commandPalette = [][2]string{
	{"New worktree", "n"},
	{"New worktree + launch claude", "N"},
	{"Attach session", "enter"},
	{"Merge into base ref", "m"},
	{"Remove worktree", "x"},
	{"Jump to next needs-input", "a"},
	{"Open in $EDITOR", "e"},
	{"Refresh git status", "r"},
	{"Keybinds & settings", "?"},
}

type Model struct {
	cfg config.Config

	worktree []Worktree
	cursor   int
	mode     mode
	tab      tabKind

	branchInput  textinput.Model
	baseInput    textinput.Model
	newFocusBase bool
	newRepo      config.Repo
	autoLaunch   bool

	confirmTarget Worktree
	confirmWhich  confirmKind

	filterInput textinput.Model
	filterQuery string

	paletteInput textinput.Model
	paletteIdx   int

	diffSession string // session name the loaded diff*/ fields belong to
	diffFiles   []gitops.FileDiff
	diffFileIdx int
	diffLines   []gitops.DiffLine
	diffLoading bool
	diffErr     error
	diffScroll  int  // line offset into the current file's rendered hunk, for mouse-wheel scrolling
	diffSplit   bool // false = unified diff, true = side-by-side split view

	activityScroll int // line offset into the activity log, for mouse-wheel scrolling

	activityLog *activity.Log
	lastChanged map[string]time.Time   // session name -> when its pane last changed
	knownState  map[string]watch.State // session name -> last classified state, to detect transitions

	paneSnapshot watch.Snapshot

	attachment    *termpty.Attachment
	attachLastEsc time.Time

	// Mouse plumbing: hits is rebuilt from scratch by every View() call and
	// consulted by the next mouse event in Update() (see hit.go's own doc
	// comment on why one frame of staleness is fine). hoverKind/hoverIdx is
	// whatever region the pointer was last found over, for hover styling;
	// lastClick* is state for sidebar-row double-click-to-attach detection.
	hits         *hitMap
	hoverKind    hitKind
	hoverIdx     int
	lastClickIdx int
	lastClickAt  time.Time

	width, height int

	status string
	err    error
	busy   bool
}

func New(cfg config.Config) Model {
	bi := textinput.New()
	bi.Placeholder = "feat/my-branch"
	bi.CharLimit = 200

	ri := textinput.New()
	ri.Placeholder = "origin/main"
	ri.CharLimit = 200

	fi := textinput.New()
	fi.Placeholder = "branch / repo / path"
	fi.CharLimit = 200

	pi := textinput.New()
	pi.Placeholder = "run a command"
	pi.CharLimit = 200

	return Model{
		cfg:          cfg,
		branchInput:  bi,
		baseInput:    ri,
		filterInput:  fi,
		paletteInput: pi,
		activityLog:  activity.NewLog(200),
		lastChanged:  make(map[string]time.Time),
		knownState:   make(map[string]watch.State),
		hits:         &hitMap{},
		hoverKind:    hitNone,
		lastClickIdx: -1,
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

type actionDoneMsg struct {
	status string
	branch string // worktree branch the action applied to, for the activity log; empty to skip logging
	tone   activity.Tone
}
type actionErrMsg struct{ err error }
type editorFinishedMsg struct{ err error }

type pollResultMsg struct {
	updates  []watch.Update
	snapshot watch.Snapshot
}

type diffFilesLoadedMsg struct {
	session string
	files   []gitops.FileDiff
	err     error
}

type diffFileLoadedMsg struct {
	session string
	path    string
	lines   []gitops.DiffLine
	err     error
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
			return actionDoneMsg{status: fmt.Sprintf("created worktree %s", branch), branch: branch, tone: activity.ToneHighlight}
		}
		name := SessionName(repo.Name, branch)
		if err := session.New(name, path, "claude"); err != nil {
			return actionErrMsg{err}
		}
		return actionDoneMsg{status: fmt.Sprintf("created %s and launched claude", branch), branch: branch, tone: activity.ToneHighlight}
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
		return actionDoneMsg{
			status: fmt.Sprintf("removed worktree %s (session killed, branch deleted)", wt.Branch),
			branch: wt.Branch, tone: activity.ToneHighlight,
		}
	}
}

func mergeCmd(wt Worktree) tea.Cmd {
	return func() tea.Msg {
		if err := gitops.Merge(wt.RepoPath, wt.Branch); err != nil {
			return actionErrMsg{err}
		}
		return actionDoneMsg{
			status: fmt.Sprintf("merged %s into %s", wt.Branch, wt.Base),
			branch: wt.Branch, tone: activity.ToneHighlight,
		}
	}
}

func loadDiffFilesCmd(wt Worktree) tea.Cmd {
	return func() tea.Msg {
		files, err := gitops.DiffFiles(wt.Path, wt.Base)
		return diffFilesLoadedMsg{session: wt.Session, files: files, err: err}
	}
}

func loadDiffFileCmd(wt Worktree, path string) tea.Cmd {
	return func() tea.Msg {
		lines, err := gitops.DiffFile(wt.Path, wt.Base, path)
		return diffFileLoadedMsg{session: wt.Session, path: path, lines: lines, err: err}
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

// applyPollUpdates folds a poll cycle's results into the model: refreshing
// each row's state/pane, noting when a pane last changed (for the session
// tab's "last changed" line), and logging state transitions to the
// activity feed. The first observation of any session only baselines
// knownState — it doesn't log, or every worktree would log a fake
// "transition" the moment wtm starts up.
func (m *Model) applyPollUpdates(updates []watch.Update) {
	byName := make(map[string]watch.Update, len(updates))
	for _, u := range updates {
		byName[u.Name] = u
	}
	for i := range m.worktree {
		u, ok := byName[m.worktree[i].Session]
		if !ok {
			continue
		}
		if prev, known := m.knownState[u.Name]; known && prev != u.State {
			m.logTransition(m.worktree[i].Branch, prev, u.State)
		}
		m.knownState[u.Name] = u.State
		if u.Pane != "" && u.Pane != m.worktree[i].LastPane {
			m.lastChanged[u.Name] = time.Now()
		}
		m.worktree[i].State = u.State
		if u.Pane != "" {
			m.worktree[i].LastPane = u.Pane
		}
	}
}

func (m *Model) logTransition(branch string, from, to watch.State) {
	var text string
	tone := activity.ToneNormal
	switch to {
	case watch.StateNeedsInput:
		text = "needs input"
		tone = activity.ToneNeedsInput
	case watch.StateRunning:
		text = fmt.Sprintf("state %s → running", stateLabel(from))
	case watch.StateIdle:
		text = fmt.Sprintf("state %s → idle", stateLabel(from))
	case watch.StateStopped:
		text = "session stopped"
	}
	if text == "" {
		return
	}
	m.activityLog.Add(branch, text, tone)
}

// Update handles all key/message routing. Attach embeds a live PTY-backed
// terminal (internal/termpty) in place of the main content pane rather than
// suspending the TUI.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.attachment != nil {
			_, mainW, _, contentH := layoutMetrics(m.width, m.height, len(needsInputRows(m.visibleWorktrees())) > 0)
			m.attachment.Resize(mainW, contentH-2)
		}
		return m, nil

	case worktreesLoadedMsg:
		m.worktree = msg.rows
		if n := len(m.visibleWorktrees()); m.cursor >= n {
			m.cursor = max(n-1, 0)
		}
		m.busy = false
		if len(msg.errs) > 0 {
			m.err = msg.errs[0]
		} else {
			m.err = nil
		}
		cmd := m.syncDiffIfNeeded()
		return m, cmd

	case actionDoneMsg:
		m.busy = false
		m.status = msg.status
		m.err = nil
		if msg.branch != "" {
			m.activityLog.Add(msg.branch, msg.status, msg.tone)
		}
		return m, loadWorktreesCmd(m.cfg)

	case actionErrMsg:
		m.busy = false
		m.err = msg.err
		return m, nil

	case termpty.StartedMsg:
		if msg.Err != nil {
			m.mode = modeList
			m.err = msg.Err
			return m, nil
		}
		m.attachment = msg.Attachment
		return m, nil

	case termpty.FrameMsg:
		return m, nil

	case termpty.ExitedMsg:
		if m.attachment != nil {
			_ = m.attachment.Close()
		}
		m.attachment = nil
		m.mode = modeList
		if msg.Err != nil {
			m.err = msg.Err
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
		m.applyPollUpdates(msg.updates)
		m.paneSnapshot = msg.snapshot
		return m, nil

	case diffFilesLoadedMsg:
		sel, ok := m.selectedWorktree()
		if !ok || sel.Session != msg.session {
			return m, nil // stale: selection moved on before this arrived
		}
		m.diffFiles = msg.files
		m.diffFileIdx = 0
		m.diffScroll = 0
		m.diffErr = msg.err
		m.diffLines = nil
		if msg.err == nil && len(msg.files) > 0 {
			m.diffLoading = true
			return m, loadDiffFileCmd(sel, msg.files[0].Path)
		}
		m.diffLoading = false
		return m, nil

	case diffFileLoadedMsg:
		sel, ok := m.selectedWorktree()
		if !ok || sel.Session != msg.session {
			return m, nil
		}
		if m.diffFileIdx >= len(m.diffFiles) || m.diffFiles[m.diffFileIdx].Path != msg.path {
			return m, nil // stale: file selection moved on before this arrived
		}
		m.diffLines = msg.lines
		m.diffErr = msg.err
		m.diffLoading = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		switch m.mode {
		case modeNewWorktree, modeConfirm, modePalette, modeHelp:
			m.mode = modeList
			return m, nil
		case modeFilter:
			m.mode = modeList
			m.filterQuery = ""
			m.filterInput.SetValue("")
			m.cursor = 0
			cmd := m.syncDiffIfNeeded()
			return m, cmd
		}
	}

	switch m.mode {
	case modeNewWorktree:
		return m.handleNewWorktreeKey(msg)
	case modeConfirm:
		return m.handleConfirmKey(msg)
	case modeFilter:
		return m.handleFilterKey(msg)
	case modePalette:
		return m.handlePaletteKey(msg)
	case modeHelp:
		return m, nil // only esc (handled above) closes it
	case modeAttached:
		return m.handleAttachedKey(msg)
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
		return m.openConfirm(confirmRemove)

	case key.Matches(msg, keys.Merge):
		return m.openConfirm(confirmMerge)

	case key.Matches(msg, keys.Attach):
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.attach(wt)

	case key.Matches(msg, keys.Tab):
		m.tab = nextTab(m.tab)
		cmd := m.syncDiffIfNeeded()
		return m, cmd

	case key.Matches(msg, keys.NextNeedsInput):
		m.jumpToNextNeedsInput()
		cmd := m.syncDiffIfNeeded()
		return m, cmd

	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filterInput.SetValue(m.filterQuery)
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
		return m, nil

	case key.Matches(msg, keys.Palette):
		m.mode = modePalette
		m.paletteInput.SetValue("")
		m.paletteInput.Focus()
		m.paletteIdx = 0
		return m, nil

	case key.Matches(msg, keys.Help):
		m.mode = modeHelp
		return m, nil

	case key.Matches(msg, keys.Edit):
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.openEditor(wt)

	case key.Matches(msg, keys.Down):
		m.cursor = nextCursor(m.visibleWorktrees(), m.cursor, 1)
		cmd := m.syncDiffIfNeeded()
		return m, cmd

	case key.Matches(msg, keys.Up):
		m.cursor = nextCursor(m.visibleWorktrees(), m.cursor, -1)
		cmd := m.syncDiffIfNeeded()
		return m, cmd
	}

	// Diff-tab-only file cycling and view toggle. Not global keybinds (no
	// sidebar equivalent to conflict with), so they're handled outside the
	// keyMap.
	if m.tab == tabDiff && len(m.diffFiles) > 0 {
		wt, ok := m.selectedWorktree()
		switch msg.String() {
		case "]":
			if ok {
				m.diffFileIdx = (m.diffFileIdx + 1) % len(m.diffFiles)
				m.diffScroll = 0
				m.diffLoading = true
				return m, loadDiffFileCmd(wt, m.diffFiles[m.diffFileIdx].Path)
			}
		case "[":
			if ok {
				m.diffFileIdx = (m.diffFileIdx - 1 + len(m.diffFiles)) % len(m.diffFiles)
				m.diffScroll = 0
				m.diffLoading = true
				return m, loadDiffFileCmd(wt, m.diffFiles[m.diffFileIdx].Path)
			}
		case "s":
			m.diffSplit = !m.diffSplit
			m.diffScroll = 0
			return m, nil
		}
	}

	return m, nil
}

func (m Model) openConfirm(kind confirmKind) (tea.Model, tea.Cmd) {
	wt, ok := m.selectedWorktree()
	if !ok {
		return m, nil
	}
	m.confirmTarget = wt
	m.confirmWhich = kind
	m.mode = modeConfirm
	return m, nil
}

// startNewWorktree opens the n/N modal. autoLaunch remembers whether N (vs
// plain n) was pressed, so submitting the form knows to launch `claude` in
// the worktree immediately after creating it.
func (m Model) startNewWorktree(autoLaunch bool) (tea.Model, tea.Cmd) {
	if len(m.cfg.Repos) == 0 {
		m.err = fmt.Errorf("no repos configured")
		return m, nil
	}
	m.newRepo = m.repoForNewWorktree()
	m.autoLaunch = autoLaunch
	m.mode = modeNewWorktree
	m.newFocusBase = false
	m.branchInput.SetValue("")
	m.baseInput.SetValue("")
	m.branchInput.Focus()
	m.baseInput.Blur()
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
// this is what the sidebar actually displays, so cursor positions and jumps
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
	if m.cursor < 0 || m.cursor >= len(rows) {
		return Worktree{}, false
	}
	return rows[m.cursor], true
}

func nextCursor(rows []Worktree, cursor, d int) int {
	n := len(rows)
	if n == 0 {
		return 0
	}
	return ((cursor+d)%n + n) % n
}

// jumpToNextNeedsInput moves the cursor to the next row (wrapping around)
// whose session is waiting on a human, and switches to the session tab so
// the reason it's waiting is immediately visible.
func (m *Model) jumpToNextNeedsInput() {
	rows := m.visibleWorktrees()
	n := len(rows)
	if n == 0 {
		return
	}
	for i := 1; i <= n; i++ {
		idx := (m.cursor + i) % n
		if rows[idx].State == watch.StateNeedsInput {
			m.cursor = idx
			m.tab = tabSession
			m.status = "jumped to " + rows[idx].Branch
			return
		}
	}
	m.status = "nothing waiting on input"
}

// syncDiffIfNeeded (re)loads the diff tab's data when it's showing and the
// selection has moved to a different worktree since the last load. It's a
// pointer method so callers must capture its returned tea.Cmd into a local
// before returning it alongside m — `return m, m.syncDiffIfNeeded()` would
// evaluate `m` (the return value) before the mutation lands, silently
// dropping the diff-loading state change.
func (m *Model) syncDiffIfNeeded() tea.Cmd {
	if m.tab != tabDiff {
		return nil
	}
	wt, ok := m.selectedWorktree()
	if !ok || m.diffSession == wt.Session {
		return nil
	}
	m.diffSession = wt.Session
	m.diffFiles = nil
	m.diffLines = nil
	m.diffFileIdx = 0
	m.diffScroll = 0
	m.diffErr = nil
	m.diffLoading = true
	return loadDiffFilesCmd(wt)
}

func (m Model) handleNewWorktreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.newFocusBase = !m.newFocusBase
		if m.newFocusBase {
			m.branchInput.Blur()
			m.baseInput.Focus()
		} else {
			m.baseInput.Blur()
			m.branchInput.Focus()
		}
		return m, nil
	case "enter":
		if m.branchInput.Value() == "" {
			return m, nil
		}
		m.mode = modeList
		m.busy = true
		branch := m.branchInput.Value()
		baseRef := m.baseInput.Value()
		return m, addWorktreeCmd(m.cfg, m.newRepo, branch, baseRef, m.autoLaunch)
	}
	var cmd tea.Cmd
	if m.newFocusBase {
		m.baseInput, cmd = m.baseInput.Update(msg)
	} else {
		m.branchInput, cmd = m.branchInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.mode = modeList
		m.busy = true
		wt := m.confirmTarget
		if m.confirmWhich == confirmMerge {
			return m, mergeCmd(wt)
		}
		return m, removeWorktreeCmd(wt)
	case "n", "N":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

// handleFilterKey applies the filter live as the user types, so the sidebar
// narrows immediately rather than waiting for a confirming Enter.
func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterQuery = m.filterInput.Value()
	m.cursor = 0
	diffCmd := m.syncDiffIfNeeded()
	return m, tea.Batch(cmd, diffCmd)
}

func (m Model) commandList() [][2]string {
	q := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	if q == "" {
		return commandPalette
	}
	var out [][2]string
	for _, c := range commandPalette {
		if strings.Contains(strings.ToLower(c[0]), q) {
			out = append(out, c)
		}
	}
	return out
}

func (m Model) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if cmds := m.commandList(); len(cmds) > 0 {
			m.paletteIdx = (m.paletteIdx - 1 + len(cmds)) % len(cmds)
		}
		return m, nil
	case "down":
		if cmds := m.commandList(); len(cmds) > 0 {
			m.paletteIdx = (m.paletteIdx + 1) % len(cmds)
		}
		return m, nil
	case "enter":
		cmds := m.commandList()
		if m.paletteIdx < len(cmds) {
			return m.runCommand(cmds[m.paletteIdx][0])
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	m.paletteIdx = 0
	return m, cmd
}

func (m Model) runCommand(label string) (tea.Model, tea.Cmd) {
	m.mode = modeList
	switch label {
	case "New worktree":
		return m.startNewWorktree(false)
	case "New worktree + launch claude":
		return m.startNewWorktree(true)
	case "Attach session":
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.attach(wt)
	case "Merge into base ref":
		return m.openConfirm(confirmMerge)
	case "Remove worktree":
		return m.openConfirm(confirmRemove)
	case "Jump to next needs-input":
		m.jumpToNextNeedsInput()
		cmd := m.syncDiffIfNeeded()
		return m, cmd
	case "Open in $EDITOR":
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.openEditor(wt)
	case "Refresh git status":
		m.busy = true
		return m, loadWorktreesCmd(m.cfg)
	case "Keybinds & settings":
		m.mode = modeHelp
		return m, nil
	}
	return m, nil
}

// attach ensures a tmux session exists for wt (creating one running `claude`
// if needed), then embeds a live PTY-backed `tmux attach-session` client in
// the main content pane (internal/termpty) instead of suspending the TUI.
func (m Model) attach(wt Worktree) (tea.Model, tea.Cmd) {
	if !session.Exists(wt.Session) {
		if err := session.New(wt.Session, wt.Path, "claude"); err != nil {
			m.err = err
			return m, nil
		}
	}
	m.mode = modeAttached
	m.tab = tabSession
	_, mainW, _, contentH := layoutMetrics(m.width, m.height, len(needsInputRows(m.visibleWorktrees())) > 0)
	return m, termpty.AttachCmd(wt.Session, mainW, contentH-2)
}

// attachFrameOrigin returns the embedded PTY frame's top-left cell in
// absolute screen coordinates: sidebar + 1-column divider to its left,
// header + optional fleet bar + tab strip + the attached view's own
// head/hint lines above it. This is the exact geometry attach() and the
// WindowSizeMsg resize handler already size the emulator against (mainW,
// contentH-2), kept in one place so a mouse event's screen coordinates can
// be translated into the emulator's own cell grid without drifting from it.
func (m Model) attachFrameOrigin() (x0, y0 int) {
	sidebarW, _, _, _ := layoutMetrics(m.width, m.height, false)
	x0 = sidebarW + 1
	y0 = 4 // header(1) + tab strip(1) + attached head(1) + hint(1)
	if len(needsInputRows(m.visibleWorktrees())) > 0 {
		y0++
	}
	return x0, y0
}

// handleAttachedKey forwards nearly every keystroke straight into the
// attached PTY. tmux's own Ctrl-b d prefix reaches the session like any
// other keystroke and "just works" for detaching, since it's a real tmux
// client on the other end; a double Escape within doubleEscWindow is a
// wtm-level backup detach for anyone who doesn't remember that prefix.
//
// The second Escape of that pair is swallowed rather than forwarded: it's
// wtm's own gesture, not input for whatever's running in the session, and
// forwarding it anyway would hand a bare Escape to programs — Claude Code
// included — that treat a quick double-Escape as their own shortcut (e.g.
// rewinding/clearing input), right as wtm detaches. A lone Escape (no
// second one following within the window) still goes through untouched.
func (m Model) handleAttachedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.attachment == nil {
		m.mode = modeList
		return m, nil
	}
	if msg.Type == tea.KeyEsc {
		now := time.Now()
		if !m.attachLastEsc.IsZero() && now.Sub(m.attachLastEsc) < doubleEscWindow {
			_ = m.attachment.Detach()
			m.attachLastEsc = time.Time{}
			return m, nil
		}
		m.attachLastEsc = now
	}
	m.attachment.Forward(msg)
	return m, nil
}

// forwardMouseToAttachment translates a mouse event's absolute screen
// coordinates into the attached PTY's own cell grid (see
// attachFrameOrigin) and hands it to the emulator — wheel scroll included,
// which is what makes the embedded terminal scrollable: wtm-managed
// sessions run with tmux's own `mouse on` (internal/session.New), so a
// forwarded wheel event drives tmux's native copy-mode/scrollback exactly
// as it would for a real terminal, with no separate scrollback of our own
// to keep in sync.
func (m Model) forwardMouseToAttachment(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.attachment == nil {
		return m, nil
	}
	x0, y0 := m.attachFrameOrigin()
	w, h := m.attachment.Size()
	msg.X = min(max(msg.X-x0, 0), max(w-1, 0))
	msg.Y = min(max(msg.Y-y0, 0), max(h-1, 0))
	m.attachment.ForwardMouse(msg)
	return m, nil
}

// handleMouse dispatches a mouse event by mode: while attached, everything
// goes straight to the PTY (see forwardMouseToAttachment) except the
// detach-hint line's own region, which is checked first so that button
// stays clickable/hoverable without ever leaking into the session;
// otherwise motion updates hover state, wheel scrolls whatever pane is
// under the pointer, and a left click/press activates whatever hitMap
// region (if any) it landed on.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode == modeAttached {
		if r, ok := m.hits.at(msg.X, msg.Y); ok && r.kind == hitDetachButton {
			m.hoverKind, m.hoverIdx = hitDetachButton, 0
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				return m.click(r)
			}
			return m, nil
		}
		if m.hoverKind == hitDetachButton {
			m.hoverKind, m.hoverIdx = hitNone, -1
		}
		return m.forwardMouseToAttachment(msg)
	}

	if msg.Action == tea.MouseActionMotion {
		if r, ok := m.hits.at(msg.X, msg.Y); ok {
			m.hoverKind, m.hoverIdx = r.kind, r.idx
		} else {
			m.hoverKind, m.hoverIdx = hitNone, -1
		}
		return m, nil
	}

	if isWheel(msg.Button) {
		return m.handleWheel(msg)
	}

	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	r, ok := m.hits.at(msg.X, msg.Y)
	if !ok {
		return m, nil
	}
	return m.click(r)
}

// isWheel reports whether a mouse button is one of the wheel directions.
// tea.MouseMsg is defined as `type MouseMsg MouseEvent`, a distinct named
// type that doesn't inherit MouseEvent's own IsWheel method.
func isWheel(b tea.MouseButton) bool {
	switch b {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown, tea.MouseButtonWheelLeft, tea.MouseButtonWheelRight:
		return true
	}
	return false
}

// handleWheel scrolls whatever pane the pointer sits over: the sidebar
// moves the cursor (matching j/k), the diff tab's file-list column cycles
// the selected file while its hunk column scrolls, and the activity tab
// scrolls its log.
func (m Model) handleWheel(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	dir := 1
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		dir = -1
	case tea.MouseButtonWheelDown:
		dir = 1
	default:
		return m, nil
	}

	sidebarW, _, _, _ := layoutMetrics(m.width, m.height, false)
	if msg.X < sidebarW {
		m.cursor = nextCursor(m.visibleWorktrees(), m.cursor, dir)
		cmd := m.syncDiffIfNeeded()
		return m, cmd
	}

	switch m.tab {
	case tabDiff:
		if msg.X < sidebarW+1+diffFileListWidth && len(m.diffFiles) > 0 {
			wt, ok := m.selectedWorktree()
			if !ok {
				return m, nil
			}
			n := len(m.diffFiles)
			m.diffFileIdx = ((m.diffFileIdx+dir)%n + n) % n
			m.diffScroll = 0
			m.diffLoading = true
			return m, loadDiffFileCmd(wt, m.diffFiles[m.diffFileIdx].Path)
		}
		m.diffScroll = max(m.diffScroll+dir*3, 0)
		return m, nil
	case tabActivity:
		m.activityScroll = max(m.activityScroll+dir*3, 0)
		return m, nil
	}
	return m, nil
}

// click activates whatever the hitMap says (x, y) landed on.
func (m Model) click(r hitRegion) (tea.Model, tea.Cmd) {
	switch r.kind {
	case hitOverlayBackdrop:
		m.mode = modeList
		return m, nil

	case hitSidebarRow:
		rows := m.visibleWorktrees()
		if r.idx < 0 || r.idx >= len(rows) {
			return m, nil
		}
		now := time.Now()
		doubleClick := m.lastClickIdx == r.idx && !m.lastClickAt.IsZero() && now.Sub(m.lastClickAt) < doubleClickWindow
		m.cursor = r.idx
		m.lastClickIdx = r.idx
		m.lastClickAt = now
		cmd := m.syncDiffIfNeeded()
		if doubleClick {
			if wt, ok := m.selectedWorktree(); ok {
				return m.attach(wt)
			}
		}
		return m, cmd

	case hitTab:
		m.tab = tabKind(r.idx)
		cmd := m.syncDiffIfNeeded()
		return m, cmd

	case hitCommandsButton:
		m.mode = modePalette
		m.paletteInput.SetValue("")
		m.paletteInput.Focus()
		m.paletteIdx = 0
		return m, nil

	case hitDiffFile:
		wt, ok := m.selectedWorktree()
		if !ok || r.idx < 0 || r.idx >= len(m.diffFiles) {
			return m, nil
		}
		m.diffFileIdx = r.idx
		m.diffScroll = 0
		m.diffLoading = true
		return m, loadDiffFileCmd(wt, m.diffFiles[r.idx].Path)

	case hitMergeButton:
		return m.openConfirm(confirmMerge)

	case hitDiscardButton:
		return m.openConfirm(confirmRemove)

	case hitSplitToggle:
		m.diffSplit = !m.diffSplit
		m.diffScroll = 0
		return m, nil

	case hitAttachButton:
		wt, ok := m.selectedWorktree()
		if !ok {
			return m, nil
		}
		return m.attach(wt)

	case hitDetachButton:
		if m.attachment != nil {
			_ = m.attachment.Detach()
		}
		return m, nil

	case hitModalPrimary:
		switch m.mode {
		case modeNewWorktree:
			return m.handleNewWorktreeKey(tea.KeyMsg{Type: tea.KeyEnter})
		case modeConfirm:
			return m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
		}
		return m, nil

	case hitModalSecondary:
		switch m.mode {
		case modeNewWorktree, modeConfirm:
			m.mode = modeList
		}
		return m, nil

	case hitPaletteRow:
		cmds := m.commandList()
		if r.idx < 0 || r.idx >= len(cmds) {
			return m, nil
		}
		return m.runCommand(cmds[r.idx][0])
	}
	return m, nil
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

func (m Model) confirmTitle() string {
	wt := m.confirmTarget
	if m.confirmWhich == confirmMerge {
		return fmt.Sprintf("Merge %s into %s?", wt.Branch, wt.Base)
	}
	return fmt.Sprintf("Remove worktree %s?", wt.Branch)
}

func (m Model) confirmSteps() []string {
	wt := m.confirmTarget
	if m.confirmWhich == confirmMerge {
		return []string{
			fmt.Sprintf("git -C %s merge --no-ff %s", wt.RepoPath, wt.Branch),
			"keeps the worktree and its session alive",
			gitLong(wt),
		}
	}
	steps := []string{
		fmt.Sprintf("tmux kill-session -t %s", wt.Session),
		fmt.Sprintf("git worktree remove %s", wt.Path),
		fmt.Sprintf("git branch -D %s", wt.Branch),
	}
	if wt.Dirty {
		steps = append(steps, "⚠ uncommitted changes will be lost")
	} else {
		steps = append(steps, "worktree is clean")
	}
	return steps
}

func (m Model) settingsRows() [][2]string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return [][2]string{
		{"worktree root", m.cfg.WorktreeRoot},
		{"agent command", "claude"},
		{"poll interval", watch.Interval.String()},
		{"repos registered", fmt.Sprintf("%d", len(m.cfg.Repos))},
		{"editor", "$EDITOR → " + editor},
		{"prune on remove", "branch + session"},
	}
}

func (m Model) newPathPreview() string {
	repo := m.newRepo.Name
	branch := m.branchInput.Value()
	if branch == "" {
		branch = "feat/my-branch"
	}
	return "path   " + filepath.Join(m.cfg.WorktreeRoot, repo, branch)
}

func (m Model) newSessionPreview() string {
	branch := m.branchInput.Value()
	if branch == "" {
		branch = "feat/my-branch"
	}
	return "tmux   " + SessionName(m.newRepo.Name, branch)
}

// View composes the full screen. Overlays (new-worktree/confirm/palette/
// help) render as a centered modal beneath the header, replacing the
// sidebar/tabs/footer rather than compositing over them — a terminal has no
// alpha-blended backdrop, and this reads as "you're in a dialog now" just
// as clearly.
func (m Model) View() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 32
	}

	m.hits.reset()
	rows := m.visibleWorktrees()
	headerRC := renderCtx{hits: m.hits, hoverKind: m.hoverKind, hoverIdx: m.hoverIdx}
	header := renderHeader(width, countFleet(m.worktree), headerRC)

	switch m.mode {
	case modeNewWorktree:
		modal, buttons := renderNewWorktreeModal(m.newRepo.Name, m.autoLaunch, m.branchInput.View(), m.baseInput.View(),
			m.newFocusBase, m.newPathPreview(), m.newSessionPreview(), m.hoverKind)
		return header + "\n" + m.placeOverlay(width, height-1, modal, buttons, 1)
	case modeConfirm:
		modal, buttons := renderConfirmModal(m.confirmTitle(), m.confirmSteps(), m.hoverKind)
		return header + "\n" + m.placeOverlay(width, height-1, modal, buttons, 1)
	case modePalette:
		hoverIdx := -1
		if m.hoverKind == hitPaletteRow {
			hoverIdx = m.hoverIdx
		}
		modal, buttons := renderPaletteModal(m.paletteInput.View(), m.commandList(), m.paletteIdx, hoverIdx)
		return header + "\n" + m.placeOverlay(width, height-1, modal, buttons, 1)
	case modeHelp:
		modal := renderHelpModal(m.settingsRows())
		return header + "\n" + m.placeOverlay(width, height-1, modal, nil, 1)
	}

	needs := needsInputRows(rows)
	var fleetBar string
	bodyY0 := 1
	if len(needs) > 0 {
		fleetBarRC := renderCtx{hits: m.hits, y0: 1, hoverKind: m.hoverKind, hoverIdx: m.hoverIdx}
		fleetBar = renderFleetBar(width, needs, fleetBarRC)
		bodyY0++
	}

	sidebarW, mainW, bodyH, contentH := layoutMetrics(width, height, fleetBar != "")

	sidebarListH := bodyH - 1 // footer line
	var filterRow string
	sidebarY0 := bodyY0
	if m.mode == modeFilter {
		filterRow = padVisible(styleAccent.Render("/ ")+m.filterInput.View(), sidebarW)
		sidebarListH--
		sidebarY0++
	}

	sidebarFooter := fmt.Sprintf("%d worktrees · %d repos", len(rows), len(m.cfg.Repos))
	if m.filterQuery != "" {
		sidebarFooter += fmt.Sprintf(" · filtered %q", m.filterQuery)
	}

	sidebarRC := renderCtx{hits: m.hits, y0: sidebarY0, hoverKind: m.hoverKind, hoverIdx: m.hoverIdx}
	var sidebarLines []string
	if filterRow != "" {
		sidebarLines = append(sidebarLines, filterRow)
	}
	sidebarLines = append(sidebarLines, renderSidebar(rows, m.cursor, sidebarW, sidebarListH, sidebarRC))
	sidebarLines = append(sidebarLines, styleDimmer.Render(padVisible(sidebarFooter, sidebarW)))
	sidebarBlock := strings.Join(sidebarLines, "\n")

	dividerLines := make([]string, bodyH)
	for i := range dividerLines {
		dividerLines[i] = styleDimmer.Render("│")
	}
	dividerBlock := strings.Join(dividerLines, "\n")

	sel, hasSel := m.selectedWorktree()
	tabRight := ""
	if hasSel {
		tabRight = sel.Path
	}
	mainX0 := sidebarW + 1
	tabRC := renderCtx{hits: m.hits, x0: mainX0, y0: bodyY0, hoverKind: m.hoverKind, hoverIdx: m.hoverIdx}
	tabStrip := renderTabStrip(m.tab, tabRight, mainW, tabRC)

	contentRC := renderCtx{hits: m.hits, x0: mainX0, y0: bodyY0 + 1, hoverKind: m.hoverKind, hoverIdx: m.hoverIdx}
	var content string
	switch {
	case !hasSel:
		content = styleDimmer.Render("no worktrees — press n to create one")
	case m.mode == modeAttached && m.attachment != nil:
		content = renderAttachedTab(sel, m.attachment.Render(), contentRC)
	case m.tab == tabDiff:
		content = renderDiffTab(sel, m.diffFiles, m.diffFileIdx, m.diffLines, m.diffLoading, m.diffErr, mainW, contentH, m.diffScroll, m.diffSplit, contentRC)
	case m.tab == tabActivity:
		content = renderActivityTab(m.activityLog.Entries(), mainW, contentH, m.activityScroll)
	default:
		content = renderSessionTab(sel, m.lastChanged[sel.Session], mainW, contentH, contentRC)
	}
	mainBlock := fitHeight(tabStrip+"\n"+content, mainW, bodyH)
	sidebarBlock = fitHeight(sidebarBlock, sidebarW, bodyH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBlock, dividerBlock, mainBlock)

	footer := renderFooter(m.footerStatus(), m.err != nil, width)

	screen := header
	if fleetBar != "" {
		screen += "\n" + fleetBar
	}
	screen += "\n" + body + "\n" + footer
	return screen
}

func (m Model) footerStatus() string {
	if m.err != nil {
		return "error: " + m.err.Error()
	}
	if m.busy {
		return "working…"
	}
	return m.status
}

func renderFooter(status string, isErr bool, width int) string {
	statusStyle := styleAccentLight
	if isErr {
		statusStyle = styleError
	}
	left := statusStyle.Render(status)
	hints := "j/k move  enter attach  tab pane  n new  N new+launch  m merge  x remove  a next needs-input  / filter  : palette  ? keys"
	right := styleDimmer.Render(hints)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		return padVisible(left, width)
	}
	return padVisible(left+strings.Repeat(" ", gap)+right, width)
}
