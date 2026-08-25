package tui

// renderAttachedTab draws the live, interactive session pane: the same
// branch/state-badge header as the read-only preview, a slim hint line, and
// the attached PTY's rendered frame filling the rest of the space. termFrame
// is already sized to the content slot (the emulator is a fixed-size cell
// grid, sized two rows shorter than the slot for this head+hint pair by
// whoever started the attachment), so unlike renderSessionTab's tailed
// capture-pane text, no clip/pad bookkeeping is needed here.
func renderAttachedTab(wt Worktree, termFrame string) string {
	head := sessionHeadLine(wt)
	hint := styleDimmer.Render("attached · typing goes to the session · esc esc or ctrl-b d to detach")
	return head + "\n" + hint + "\n" + termFrame
}
