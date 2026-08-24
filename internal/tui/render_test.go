package tui

import (
	"testing"
	"time"
)

func TestClipPad(t *testing.T) {
	if got := clipPad("hi", 5); got != "hi   " {
		t.Errorf("expected padded 'hi   ', got %q", got)
	}
	if got := clipPad("hello world", 5); got != "hell…" {
		t.Errorf("expected truncated 'hell…', got %q", got)
	}
	if got := clipPad("hello", 5); got != "hello" {
		t.Errorf("expected exact-fit unchanged, got %q", got)
	}
}

func TestClipLeft(t *testing.T) {
	got := clipLeft("src/payments/tap.ts", 10)
	if len([]rune(got)) != 10 {
		t.Fatalf("expected exactly 10 runes, got %d (%q)", len([]rune(got)), got)
	}
	if got[len(got)-6:] != "tap.ts" {
		t.Errorf("expected the filename preserved at the end, got %q", got)
	}
	if got := clipLeft("short.go", 20); got != "short.go" {
		t.Errorf("expected short strings unchanged, got %q", got)
	}
}

func TestTailLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}
	if got := tailLines(lines, 2); len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Errorf("expected last 2 lines [c d], got %v", got)
	}
	if got := tailLines(lines, 10); len(got) != 4 {
		t.Errorf("expected all lines when n exceeds length, got %v", got)
	}
	if got := tailLines(lines, 0); got != nil {
		t.Errorf("expected nil for n<=0, got %v", got)
	}
}

func TestHumanizeAgo(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m"},
		{75 * time.Minute, "1h 15m"},
		{2 * time.Hour, "2h"},
	}
	for _, c := range cases {
		if got := humanizeAgo(c.d); got != c.want {
			t.Errorf("humanizeAgo(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
