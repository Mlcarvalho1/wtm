package watch

import (
	"strings"
	"testing"

	"github.com/Mlcarvalho1/wtm/internal/session"
)

func uniqueName(t *testing.T) string {
	t.Helper()
	return "wtm-watch-test-" + strings.ReplaceAll(t.Name(), "/", "-")
}

func TestPoll_StoppedSession(t *testing.T) {
	updates, next := Poll([]string{uniqueName(t) + "-nonexistent"}, Snapshot{})
	if len(updates) != 1 || updates[0].State != StateStopped {
		t.Fatalf("expected single Stopped update, got %+v", updates)
	}
	if len(next) != 0 {
		t.Fatalf("expected empty snapshot for stopped session, got %+v", next)
	}
}

func TestPoll_FirstSightingIsRunning(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()
	if err := session.New(name, dir, "sh -c \"echo hi; sleep 100\""); err != nil {
		t.Fatalf("session.New: %v", err)
	}
	defer session.Kill(name)

	updates, next := Poll([]string{name}, Snapshot{})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].State != StateRunning {
		t.Errorf("expected first sighting to classify as Running, got %v", updates[0].State)
	}
	if _, ok := next[name]; !ok {
		t.Errorf("expected snapshot to record pane for %q", name)
	}
}

func TestPoll_UnchangedPaneIsIdle(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()
	if err := session.New(name, dir, "sh -c \"echo static; sleep 100\""); err != nil {
		t.Fatalf("session.New: %v", err)
	}
	defer session.Kill(name)

	updates, next := Poll([]string{name}, Snapshot{})
	pane := updates[0].Pane

	updates, _ = Poll([]string{name}, next)
	if updates[0].State != StateIdle {
		t.Errorf("expected unchanged pane to classify as Idle, got %v (pane=%q)", updates[0].State, pane)
	}
}

func TestPoll_UnchangedWithWaitingPatternIsNeedsInput(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()
	if err := session.New(name, dir, "sh -c \"echo 'Do you want to proceed?'; sleep 100\""); err != nil {
		t.Fatalf("session.New: %v", err)
	}
	defer session.Kill(name)

	_, next := Poll([]string{name}, Snapshot{})
	updates, _ := Poll([]string{name}, next)
	if updates[0].State != StateNeedsInput {
		t.Errorf("expected unchanged waiting-pattern pane to classify as NeedsInput, got %v", updates[0].State)
	}
}

func TestPoll_KilledBetweenCallsBecomesStopped(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()
	if err := session.New(name, dir, "sleep 100"); err != nil {
		t.Fatalf("session.New: %v", err)
	}

	_, next := Poll([]string{name}, Snapshot{})
	if err := session.Kill(name); err != nil {
		t.Fatalf("Kill: %v", err)
	}

	updates, _ := Poll([]string{name}, next)
	if updates[0].State != StateStopped {
		t.Errorf("expected killed session to classify as Stopped, got %v", updates[0].State)
	}
}

func TestIsWaiting(t *testing.T) {
	cases := map[string]bool{
		"":                                     false,
		"regular claude output, still typing…": false,
		"? for shortcuts":                      true,
		"Do you want to proceed?":              true,
		"Is this a project you created or one you trust?": true,
	}
	for pane, want := range cases {
		if got := IsWaiting(pane); got != want {
			t.Errorf("IsWaiting(%q) = %v, want %v", pane, got, want)
		}
	}
}
