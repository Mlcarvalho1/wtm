package tui

import (
	"testing"
	"time"

	"github.com/Mlcarvalho1/wtm/internal/activity"
	"github.com/Mlcarvalho1/wtm/internal/watch"
)

func TestVisibleWorktrees_Filter(t *testing.T) {
	m := Model{worktree: []Worktree{
		{RepoName: "amigo-app", Branch: "feat/tap-to-pay"},
		{RepoName: "amigo-app", Branch: "fix/webhook-retry"},
		{RepoName: "wtm", Branch: "feat/preview-pane"},
	}}

	m.filterQuery = "amigo"
	if got := len(m.visibleWorktrees()); got != 2 {
		t.Errorf("expected 2 rows for 'amigo', got %d", got)
	}

	m.filterQuery = "preview"
	rows := m.visibleWorktrees()
	if len(rows) != 1 || rows[0].Branch != "feat/preview-pane" {
		t.Errorf("expected only feat/preview-pane, got %+v", rows)
	}

	m.filterQuery = ""
	if got := len(m.visibleWorktrees()); got != 3 {
		t.Errorf("expected all 3 rows with no filter, got %d", got)
	}
}

func TestNextCursor_Wraps(t *testing.T) {
	rows := make([]Worktree, 3)
	if got := nextCursor(rows, 0, -1); got != 2 {
		t.Errorf("expected wrap to 2, got %d", got)
	}
	if got := nextCursor(rows, 2, 1); got != 0 {
		t.Errorf("expected wrap to 0, got %d", got)
	}
	if got := nextCursor(nil, 0, 1); got != 0 {
		t.Errorf("expected 0 for empty rows, got %d", got)
	}
}

func TestJumpToNextNeedsInput(t *testing.T) {
	m := Model{
		worktree: []Worktree{
			{Branch: "a", State: watch.StateRunning},
			{Branch: "b", State: watch.StateNeedsInput},
			{Branch: "c", State: watch.StateIdle},
		},
		tab: tabDiff,
	}
	m.jumpToNextNeedsInput()
	if m.cursor != 1 {
		t.Errorf("expected cursor at index 1, got %d", m.cursor)
	}
	if m.tab != tabSession {
		t.Errorf("expected jump to switch to session tab, got %v", m.tab)
	}
}

func TestJumpToNextNeedsInput_NoneWaiting(t *testing.T) {
	m := Model{worktree: []Worktree{{Branch: "a", State: watch.StateRunning}}}
	m.jumpToNextNeedsInput()
	if m.status != "nothing waiting on input" {
		t.Errorf("expected status message, got %q", m.status)
	}
}

func TestGitCellAndGitLong(t *testing.T) {
	clean := Worktree{}
	if gitCell(clean) != "clean" {
		t.Errorf("expected clean, got %q", gitCell(clean))
	}
	if gitLong(clean) != "clean, in sync" {
		t.Errorf("expected clean in sync, got %q", gitLong(clean))
	}

	dirty := Worktree{Dirty: true, Ahead: 2, Behind: 1}
	if got := gitCell(dirty); got != "*↑2↓1" {
		t.Errorf("expected *↑2↓1, got %q", got)
	}
	if got := gitLong(dirty); got != "uncommitted changes · 2 ahead · 1 behind" {
		t.Errorf("unexpected gitLong: %q", got)
	}
}

func TestApplyPollUpdates_LogsTransitionsNotFirstObservation(t *testing.T) {
	m := &Model{
		worktree:    []Worktree{{Branch: "feat/x", Session: "wtm-feat-x", State: watch.StateStopped}},
		activityLog: activity.NewLog(10),
		lastChanged: map[string]time.Time{},
		knownState:  map[string]watch.State{},
	}

	m.applyPollUpdates([]watch.Update{{Name: "wtm-feat-x", State: watch.StateRunning, Pane: "hello"}})
	if len(m.activityLog.Entries()) != 0 {
		t.Fatalf("expected no log entry on first observation, got %+v", m.activityLog.Entries())
	}
	if m.worktree[0].State != watch.StateRunning {
		t.Errorf("expected state updated to running")
	}
	if _, ok := m.lastChanged["wtm-feat-x"]; !ok {
		t.Errorf("expected lastChanged to be set after a pane change")
	}

	m.applyPollUpdates([]watch.Update{{Name: "wtm-feat-x", State: watch.StateNeedsInput, Pane: "hello"}})
	entries := m.activityLog.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry after a real transition, got %d: %+v", len(entries), entries)
	}
	if entries[0].Branch != "feat/x" || entries[0].Tone != activity.ToneNeedsInput {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestSyncDiffIfNeeded_OnlyWhenDiffTabAndSelectionChanged(t *testing.T) {
	m := &Model{
		worktree: []Worktree{{Branch: "feat/x", Session: "wtm-feat-x", Base: "main"}},
		tab:      tabSession,
	}
	if cmd := m.syncDiffIfNeeded(); cmd != nil {
		t.Errorf("expected no reload while on session tab")
	}

	m.tab = tabDiff
	if cmd := m.syncDiffIfNeeded(); cmd == nil {
		t.Errorf("expected a reload command when switching to diff tab")
	}
	if m.diffSession != "wtm-feat-x" {
		t.Errorf("expected diffSession set to selected session, got %q", m.diffSession)
	}

	if cmd := m.syncDiffIfNeeded(); cmd != nil {
		t.Errorf("expected no reload when selection hasn't changed")
	}
}
