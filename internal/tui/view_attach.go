package tui

import "github.com/charmbracelet/lipgloss"

// renderAttachedTab draws the live, interactive session pane: the same
// branch/state-badge header as the read-only preview, a slim hint line
// ending in a clickable/hoverable "detach" button (matching the design's
// explicit detach button, since esc-esc/ctrl-b d alone aren't obvious), and
// the attached PTY's rendered frame filling the rest of the space.
// termFrame is already sized to the content slot (the emulator is a
// fixed-size cell grid, sized two rows shorter than the slot for this
// head+hint pair by whoever started the attachment), so unlike
// renderSessionTab's tailed capture-pane text, no clip/pad bookkeeping is
// needed here.
func renderAttachedTab(wt Worktree, termFrame string, rc renderCtx) string {
	head := sessionHeadLine(wt)

	hintRC := rc.translate(0, 1)
	hintStyle := styleDimmer
	if hintRC.hovered(hitDetachButton, 0) {
		hintStyle = lipgloss.NewStyle().Foreground(colorAccentLight).Background(colorHoverAccentBg)
	}
	const hintText = "attached · typing goes to the session · esc esc or ctrl-b d to detach"
	hint := hintStyle.Render(hintText)
	hintRC.addHit(hitDetachButton, 0, 0, 0, lipgloss.Width(hintText), 1)

	return head + "\n" + hint + "\n" + termFrame
}
