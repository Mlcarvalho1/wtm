package watch

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Mlcarvalho1/wtm/internal/session"
)

// Interval is how often the fleet-status ticker polls live sessions.
const Interval = 2 * time.Second

// TickMsg fires the poll loop. Callers should re-issue Tick() after handling
// it to keep the loop going — it does not reschedule itself.
type TickMsg struct{}

// Tick schedules the next poll tick as a tea.Cmd.
func Tick() tea.Cmd {
	return tea.Tick(Interval, func(time.Time) tea.Msg { return TickMsg{} })
}

// Update is one session's freshly classified state from a single Poll call.
type Update struct {
	Name  string
	State State
	Pane  string
}

// Snapshot holds the last captured pane text per session, to diff against on
// the next Poll call.
type Snapshot map[string]string

// Poll captures each named session's current pane, diffs it against prev,
// and classifies its state. It returns the per-session updates plus the new
// snapshot to pass as prev on the next call.
//
// Poll is a pure function of its inputs — it holds no state of its own — so
// it's safe to run inside a tea.Cmd (bubbletea's own goroutine) without any
// risk of racing the render loop; all mutable state lives in the model.
func Poll(names []string, prev Snapshot) ([]Update, Snapshot) {
	next := make(Snapshot, len(names))
	updates := make([]Update, 0, len(names))

	for _, name := range names {
		if !session.Exists(name) {
			updates = append(updates, Update{Name: name, State: StateStopped})
			continue
		}

		pane, err := session.CapturePane(name)
		if err != nil {
			updates = append(updates, Update{Name: name, State: StateStopped})
			continue
		}
		next[name] = pane

		prevPane, known := prev[name]
		var state State
		switch {
		case known && pane == prevPane && IsWaiting(pane):
			state = StateNeedsInput
		case known && pane == prevPane:
			state = StateIdle
		default:
			state = StateRunning
		}
		updates = append(updates, Update{Name: name, State: state, Pane: pane})
	}

	return updates, next
}
