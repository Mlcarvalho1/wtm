package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/activity"
)

// renderActivityTab draws the fleet-wide activity feed, most recent first.
func renderActivityTab(entries []activity.Entry, width, height int) string {
	header := styleKicker.Render("FLEET ACTIVITY — ALL REPOS")
	if len(entries) == 0 {
		return header + "\n\n" + styleDimmer.Render("nothing yet")
	}

	timeW, branchW := 8, 22
	textW := max(width-timeW-branchW-4, 1)

	lines := make([]string, 0, len(entries)+2)
	lines = append(lines, header, "")
	for _, e := range entries {
		if len(lines) >= height {
			break
		}
		textColor := lipgloss.NewStyle().Foreground(colorDim)
		switch e.Tone {
		case activity.ToneHighlight:
			textColor = lipgloss.NewStyle().Foreground(colorAccentLight)
		case activity.ToneNeedsInput:
			textColor = lipgloss.NewStyle().Foreground(colorText)
		}
		line := styleDimmer.Render(clipPad(e.Time.Format("15:04"), timeW)) + " " +
			lipgloss.NewStyle().Foreground(colorDim).Render(clipPad(e.Branch, branchW)) + " " +
			textColor.Render(clipPad(e.Text, textW))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
