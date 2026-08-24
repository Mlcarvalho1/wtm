package watch

import "strings"

// waitingPatterns are substrings that show up in the Claude Code CLI's
// terminal output when it is sitting still waiting on a human: either its
// normal input prompt, or a permission/confirmation dialog. Kept in its own
// file since it needs tuning against real `claude` CLI output over time.
var waitingPatterns = []string{
	"? for shortcuts",
	"Do you want to proceed",
	"Do you want to make this edit",
	"Do you want to create",
	"Do you want to run this command",
	"No, and tell Claude what to do differently",
	"Yes, and don't ask again",
}

// IsWaiting reports whether pane looks like it's sitting at a prompt that
// needs a human response, based on the tuned waitingPatterns list above.
func IsWaiting(pane string) bool {
	for _, p := range waitingPatterns {
		if strings.Contains(pane, p) {
			return true
		}
	}
	return false
}
