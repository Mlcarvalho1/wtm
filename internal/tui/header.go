package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/watch"
)

// fleetCounts tallies how many worktrees sit in each fleet-status bucket,
// for the header bar's "N running · N need input · ..." readout.
type fleetCounts struct {
	running, needs, idle, stopped int
}

func countFleet(rows []Worktree) fleetCounts {
	var c fleetCounts
	for _, w := range rows {
		switch w.State {
		case watch.StateRunning:
			c.running++
		case watch.StateNeedsInput:
			c.needs++
		case watch.StateIdle:
			c.idle++
		default:
			c.stopped++
		}
	}
	return c
}

// renderHeader draws the top bar: "wtm" title, fleet counters, and the
// command-palette hint — itself a clickable/hoverable button matching the
// design's ": commands" button.
func renderHeader(width int, c fleetCounts, rc renderCtx) string {
	title := styleTitle.Render("wtm") + "  " + styleSubtitle.Render("worktree · agent manager")

	needsStyle := lipgloss.NewStyle().Foreground(colorText).Blink(c.needs > 0)
	counters := lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d running", c.running)) +
		"   " + needsStyle.Render(fmt.Sprintf("%d need input", c.needs)) +
		"   " + lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d idle", c.idle)) +
		"   " + lipgloss.NewStyle().Foreground(colorFaint).Render(fmt.Sprintf("%d stopped", c.stopped))

	const paletteText = ": commands"
	paletteStyle := lipgloss.NewStyle().Foreground(colorDim)
	if rc.hovered(hitCommandsButton, 0) {
		paletteStyle = lipgloss.NewStyle().Foreground(colorAccentLight)
	}
	palette := paletteStyle.Render(paletteText)

	left := title
	right := counters + "   " + palette
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	buttonX0 := gap + lipgloss.Width(left) + lipgloss.Width(counters) + 3
	rc.addHit(hitCommandsButton, 0, buttonX0, 0, buttonX0+len(paletteText), 1)
	return padVisible(left+strings.Repeat(" ", gap)+right, width)
}

// needsInputRow is one chip in the "awaiting you" bar.
type needsInputRow struct {
	Branch string
	Idx    int // index into the visible worktree list, for jumping the cursor
}

func needsInputRows(rows []Worktree) []needsInputRow {
	var out []needsInputRow
	for i, w := range rows {
		if w.State == watch.StateNeedsInput {
			out = append(out, needsInputRow{Branch: w.Branch, Idx: i})
		}
	}
	return out
}

// renderFleetBar draws the "awaiting you" strip listing every worktree
// currently needing human input, when any exist. Each chip is clickable
// (selects that worktree, same as clicking its sidebar row — so it shares
// the hitSidebarRow region kind keyed by the same visible-list index) and
// highlights on hover.
func renderFleetBar(width int, rows []needsInputRow, rc renderCtx) string {
	label := lipgloss.NewStyle().Foreground(colorAccentLight).Bold(true).Render("AWAITING YOU")
	var chips []string
	x := lipgloss.Width(label) + 2
	for _, r := range rows {
		style := lipgloss.NewStyle().Foreground(colorText)
		dotStyle := style.Blink(true)
		if rc.hovered(hitSidebarRow, r.Idx) {
			style = style.Background(colorHoverAccentBg)
			dotStyle = dotStyle.Background(colorHoverAccentBg)
		}
		chip := style.Render("[") + dotStyle.Render("●") + style.Render(" "+r.Branch+"]")
		chips = append(chips, chip)
		chipW := lipgloss.Width(r.Branch) + 4
		rc.addHit(hitSidebarRow, r.Idx, x, 0, x+chipW, 1)
		x += chipW + 2
	}
	hint := lipgloss.NewStyle().Foreground(colorDim).Render("a — jump to next")

	line := label + "  " + strings.Join(chips, "  ")
	gap := width - lipgloss.Width(line) - lipgloss.Width(hint)
	if gap < 1 {
		gap = 1
	}
	return padVisible(line+strings.Repeat(" ", gap)+hint, width)
}

// renderTabStrip draws the session/diff/activity tab row, right-aligned
// with the selected worktree's path and a "tab to cycle" hint. Each label is
// clickable (switches to that tab) and highlights on hover.
func renderTabStrip(active tabKind, rightHint string, width int, rc renderCtx) string {
	tabs := []struct {
		kind  tabKind
		label string
	}{{tabSession, "session"}, {tabDiff, "diff"}, {tabActivity, "activity"}}

	var parts []string
	x := 0
	for _, t := range tabs {
		style := styleTabInactive
		switch {
		case t.kind == active:
			style = styleTabActive
		case rc.hovered(hitTab, int(t.kind)):
			style = styleTabHover
		}
		parts = append(parts, style.Render(t.label))
		rc.addHit(hitTab, int(t.kind), x, 0, x+len(t.label), 1)
		x += len(t.label) + 3 // gap matches strings.Join(parts, "   ") below
	}
	left := strings.Join(parts, "   ")

	right := rightHint
	if right != "" {
		right += "   "
	}
	right = styleDimmer.Render(right + "tab to cycle")

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return padVisible(left+strings.Repeat(" ", gap)+right, width)
}
