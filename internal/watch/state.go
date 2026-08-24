// Package watch classifies each tmux session's live state by polling
// tmux capture-pane and diffing against the previous capture — no PTY
// buffering, no ANSI parsing, just a string diff per tick.
package watch

// State is the fleet-status classification of one worktree's tmux session.
type State int

const (
	StateStopped State = iota
	StateIdle
	StateRunning
	StateNeedsInput
)

func (s State) String() string {
	switch s {
	case StateRunning:
		return "Running"
	case StateNeedsInput:
		return "Needs Input"
	case StateIdle:
		return "Idle"
	default:
		return "Stopped"
	}
}

func (s State) Icon() string {
	switch s {
	case StateRunning:
		return "🟢"
	case StateNeedsInput:
		return "🟡"
	case StateIdle:
		return "⚪"
	default:
		return "⚫"
	}
}
