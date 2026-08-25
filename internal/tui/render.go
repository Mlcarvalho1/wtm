package tui

import "strings"

// clipPad forces s to exactly width runes: truncated with an ellipsis if
// longer, space-padded if shorter, so every line in a bordered box lines up.
// Used anywhere we render plain (non-lipgloss-measured) fixed-width text —
// lipgloss's own Width() re-wraps long lines to fit, which would mangle
// box-drawing output like Claude Code's own boxed UI in the pane preview.
func clipPad(s string, width int) string {
	if width <= 0 {
		return ""
	}
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

// clipLeft is clipPad's mirror for text where the tail matters more than the
// head — file paths, where the filename at the end is the identifying part
// (the design does this with `direction: rtl` on the file-list cells; a
// terminal has no such trick, so this does it by hand).
func clipLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 1 {
		return string(r[len(r)-width:])
	}
	return "…" + string(r[len(r)-(width-1):])
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

// fitHeight forces a multi-line block to exactly height lines: padded with
// blank width-wide lines if shorter, hard-truncated if taller. Per-tab
// renderers each do their own height bookkeeping to fit their slice of the
// screen, which is easy to get one line off in any single one of them —
// this is the backstop that keeps a stray off-by-one from misaligning the
// whole frame (and, without an alt-screen, scrolling the header off the
// top of the terminal) instead of just clipping that one panel.
func fitHeight(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

// dot renders a single-character status indicator. Filled solid for a color,
// hollow "○" for the stopped state's outline-only look in the design.
func dot(filled bool) string {
	if filled {
		return "●"
	}
	return "○"
}

// layoutMetrics computes the sidebar/main-column pixel budget from a raw
// terminal size. View() uses it to lay out each frame; the attach flow uses
// the exact same numbers to size the embedded PTY, so the two can never
// drift apart the way hand-duplicated math eventually would (see fitHeight's
// own doc comment on that failure mode).
func layoutMetrics(width, height int, showFleetBar bool) (sidebarW, mainW, bodyH, contentH int) {
	headerH, footerH := 1, 1
	fleetH := 0
	if showFleetBar {
		fleetH = 1
	}
	bodyH = max(height-headerH-fleetH-footerH, 3)

	sidebarW = width * 3 / 10
	sidebarW = min(max(sidebarW, 24), 44)
	if sidebarW > width-20 {
		sidebarW = max(width-20, 10)
	}
	mainW = max(width-sidebarW-1, 10)
	contentH = max(bodyH-1, 2)
	return sidebarW, mainW, bodyH, contentH
}
