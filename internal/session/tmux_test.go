package session

import (
	"strings"
	"testing"
	"time"
)

// uniqueName avoids collisions with any sessions already running on the
// developer's tmux server. Tests must not assume they own the whole server.
func uniqueName(t *testing.T) string {
	t.Helper()
	return "wtm-test-" + strings.ReplaceAll(t.Name(), "/", "-")
}

func TestNewExistsKill(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()

	if Exists(name) {
		t.Fatalf("session %q should not exist yet", name)
	}

	if err := New(name, dir, "sleep 100"); err != nil {
		t.Fatalf("New: %v", err)
	}
	defer Kill(name)

	if !Exists(name) {
		t.Fatalf("expected session %q to exist after New", name)
	}

	if err := Kill(name); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if Exists(name) {
		t.Fatalf("expected session %q to be gone after Kill", name)
	}

	// Kill on an already-gone session must be a no-op, not an error.
	if err := Kill(name); err != nil {
		t.Fatalf("Kill on nonexistent session should be no-op, got: %v", err)
	}
}

func TestNew_DuplicateFails(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()

	if err := New(name, dir, "sleep 100"); err != nil {
		t.Fatalf("New: %v", err)
	}
	defer Kill(name)

	if err := New(name, dir, "sleep 100"); err == nil {
		t.Fatalf("expected New to fail on duplicate session name")
	}
}

func TestList(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()

	if err := New(name, dir, "sleep 100"); err != nil {
		t.Fatalf("New: %v", err)
	}
	defer Kill(name)

	sessions, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, s := range sessions {
		if s == name {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %q in session list %v", name, sessions)
	}
}

func TestCapturePane(t *testing.T) {
	name := uniqueName(t)
	dir := t.TempDir()

	if err := New(name, dir, "sh -c \"echo hello-from-pane; sleep 100\""); err != nil {
		t.Fatalf("New: %v", err)
	}
	defer Kill(name)

	// give the shell a moment to print before we capture
	var out string
	for range 20 {
		var err error
		out, err = CapturePane(name)
		if err != nil {
			t.Fatalf("CapturePane: %v", err)
		}
		if strings.Contains(out, "hello-from-pane") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !strings.Contains(out, "hello-from-pane") {
		t.Fatalf("expected pane output to contain %q, got %q", "hello-from-pane", out)
	}
}

func TestCapturePane_NonexistentSession(t *testing.T) {
	if _, err := CapturePane(uniqueName(t) + "-does-not-exist"); err == nil {
		t.Fatalf("expected error capturing pane of nonexistent session")
	}
}
