package activity

import "testing"

func TestLog_MostRecentFirst(t *testing.T) {
	l := NewLog(10)
	l.Add("feat/a", "worktree created", ToneHighlight)
	l.Add("feat/b", "state running -> idle", ToneNormal)
	l.Add("feat/c", "needs input", ToneNeedsInput)

	entries := l.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Branch != "feat/c" || entries[0].Tone != ToneNeedsInput {
		t.Errorf("expected most recent entry first, got %+v", entries[0])
	}
	if entries[2].Branch != "feat/a" {
		t.Errorf("expected oldest entry last, got %+v", entries[2])
	}
}

func TestLog_EvictsOldest(t *testing.T) {
	l := NewLog(2)
	l.Add("a", "one", ToneNormal)
	l.Add("b", "two", ToneNormal)
	l.Add("c", "three", ToneNormal)

	entries := l.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected capacity-capped at 2 entries, got %d", len(entries))
	}
	if entries[0].Branch != "c" || entries[1].Branch != "b" {
		t.Errorf("expected [c, b], got [%s, %s]", entries[0].Branch, entries[1].Branch)
	}
}

func TestNewLog_NonPositiveCapacity(t *testing.T) {
	l := NewLog(0)
	l.Add("a", "one", ToneNormal)
	l.Add("b", "two", ToneNormal)
	if len(l.Entries()) != 1 {
		t.Errorf("expected capacity clamped to 1")
	}
}
