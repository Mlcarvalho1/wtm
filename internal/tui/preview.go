package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var previewBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240")).
	Padding(0, 1)

const previewFrameWidth = 4  // 1 border + 1 padding, each side
const previewFrameHeight = 2 // top/bottom border, no vertical padding

// renderPreview shows the tail of wt's last captured tmux pane — plain text,
// not a terminal emulation, per the plan's "no PTY buffer" design.
//
// Every line is clipped/padded to an exact rune width before it ever reaches
// lipgloss: lipgloss's own Width() re-wraps long lines to fit, which mangles
// box-drawing output like Claude Code's own boxed UI. Clipping ourselves
// keeps this a plain tail, not a reflow.
func renderPreview(wt Worktree, width, height int) string {
	innerWidth := max(width-previewFrameWidth, 1)
	innerHeight := max(height-previewFrameHeight, 1)

	lines := make([]string, 0, innerHeight)
	lines = append(lines, clipPad(fmt.Sprintf("%s  (%s)", wt.Branch, wt.Session), innerWidth))

	paneHeight := innerHeight - 1
	var paneLines []string
	if wt.LastPane == "" {
		paneLines = []string{"(no session)"}
	} else {
		paneLines = tailLines(strings.Split(strings.TrimRight(wt.LastPane, "\n"), "\n"), paneHeight)
	}
	for _, l := range paneLines {
		lines = append(lines, clipPad(l, innerWidth))
	}
	for len(lines) < innerHeight {
		lines = append(lines, clipPad("", innerWidth))
	}

	return previewBorder.Render(strings.Join(lines, "\n"))
}

// clipPad forces s to exactly width runes: truncated with an ellipsis if
// longer, space-padded if shorter, so every line in the box lines up.
func clipPad(s string, width int) string {
	r := []rune(s)
	switch {
	case len(r) == width:
		return s
	case len(r) < width:
		return s + strings.Repeat(" ", width-len(r))
	case width <= 1:
		return string(r[:width])
	default:
		return string(r[:width-1]) + "…"
	}
}

func tailLines(lines []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
