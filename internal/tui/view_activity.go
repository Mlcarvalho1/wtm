package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/activity"
)

// renderActivityTab draws the fleet-wide activity feed, most recent first.
// scroll is a line offset into entries, clamped internally to what height
// actually allows — mirroring the diff tab's hunk scrolling, so a wheel
// notch over either tab behaves the same way.
func renderActivityTab(entries []activity.Entry, width, height, scroll int) string {
	header := styleKicker.Render("FLEET ACTIVITY — ALL REPOS")
	if len(entries) == 0 {
		return header + "\n\n" + styleDimmer.Render("nothing yet")
	}

	timeW, branchW := 8, 22
	textW := max(width-timeW-branchW-4, 1)
	listH := max(height-2, 1)

	maxScroll := max(len(entries)-listH, 0)
	scroll = min(max(scroll, 0), maxScroll)
	visible := entries[scroll:min(scroll+listH, len(entries))]

	lines := make([]string, 0, len(visible)+2)
	lines = append(lines, header, "")
	for _, e := range visible {
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
