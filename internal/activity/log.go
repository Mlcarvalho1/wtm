// Package activity keeps an in-memory, session-lifetime feed of fleet
// events (worktree created, session state transitions, merges, removals)
// for the TUI's activity tab. Per wtm's "no persistent state" design
// principle, none of this survives a restart — it's derived live like
// everything else, just accumulated in memory instead of re-derived from
// git/tmux on every poll.
package activity

import "time"

// Tone is a coarse coloring hint for an Entry, left for the TUI to map to
// actual colors so this package stays presentation-agnostic.
type Tone int

const (
	ToneNormal     Tone = iota
	ToneHighlight       // created/launched/merged — a change actually happened
	ToneNeedsInput      // the fleet needs a human
)

// Entry is one line of fleet activity.
type Entry struct {
	Time   time.Time
	Branch string
	Text   string
	Tone   Tone
}

// Log is a fixed-capacity, most-recent-first feed of Entries.
type Log struct {
	entries []Entry
	cap     int
}

// NewLog creates a Log holding at most capacity entries, discarding the
// oldest once full.
func NewLog(capacity int) *Log {
	if capacity <= 0 {
		capacity = 1
	}
	return &Log{cap: capacity}
}

// Add appends a new entry timestamped now, evicting the oldest entry if the
// log is at capacity.
func (l *Log) Add(branch, text string, tone Tone) {
	l.entries = append(l.entries, Entry{Time: time.Now(), Branch: branch, Text: text, Tone: tone})
	if len(l.entries) > l.cap {
		l.entries = l.entries[len(l.entries)-l.cap:]
	}
}

// Entries returns the log's entries, most recent first.
func (l *Log) Entries() []Entry {
	out := make([]Entry, len(l.entries))
	for i, e := range l.entries {
		out[len(l.entries)-1-i] = e
	}
	return out
}
