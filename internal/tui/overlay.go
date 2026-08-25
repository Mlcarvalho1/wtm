package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// centerOverlay places modal in the middle of a width x height canvas — the
// terminal-appropriate stand-in for the design's dimmed full-screen
// backdrop, which a terminal has no alpha-blending to render.
func centerOverlay(width, height int, modal string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

// placeOverlay is centerOverlay plus the hit-map bookkeeping: a full-canvas
// backdrop region (clicking it closes the overlay, matching the design's
// onClick={closeOverlay}) registered first, then modal's own buttons (given
// in its local coordinates) on top of it, translated to whatever absolute
// screen position lipgloss.Place actually renders the modal at — computed
// with centerOffsets rather than a second, possibly-drifting copy of that
// centering math. yOffset accounts for anything the caller draws above this
// canvas (View() prepends one header line before calling this).
func (m Model) placeOverlay(width, height int, modal string, buttons []hitRegion, yOffset int) string {
	boxW := lipgloss.Width(modal)
	boxH := strings.Count(modal, "\n") + 1
	ox, oy := centerOffsets(width, height, boxW, boxH)
	oy += yOffset
	m.hits.add(hitOverlayBackdrop, 0, 0, yOffset, width, height+yOffset)
	for _, b := range buttons {
		m.hits.add(b.kind, b.idx, ox+b.x0, oy+b.y0, ox+b.x1, oy+b.y1)
	}
	return centerOverlay(width, height, modal)
}

// modalContentOrigin returns where a styleModal-wrapped body's own (0,0)
// lands relative to the modal's own rendered top-left — its border and
// padding, read directly off the style so this can never drift out of sync
// with styleModal's own definition the way a hand-copied constant could.
func modalContentOrigin() (x, y int) {
	return styleModal.GetMarginLeft() + styleModal.GetBorderLeftSize() + styleModal.GetPaddingLeft(),
		styleModal.GetMarginTop() + styleModal.GetBorderTopSize() + styleModal.GetPaddingTop()
}

// renderNewWorktreeModal draws the combined branch+base-ref form (tab moves
// between fields; both are visible at once, unlike the CLI's old two-step
// sequential prompts). The returned regions are in the modal's own local
// coordinates (0,0 at its top-left); the caller (Model.View) translates them
// to wherever centerOverlay actually places the modal on screen.
func renderNewWorktreeModal(repoName string, launch bool, branchView, baseView string, focusBase bool, pathPreview, sessionPreview string, hover hitKind) (string, []hitRegion) {
	title := "new worktree"
	if launch {
		title += " + launch claude"
	}
	sub := fmt.Sprintf("repo %s · %s", repoName, launchNote(launch))

	branchLabel := styleKicker.Render("BRANCH")
	baseLabel := styleKicker.Render("BASE REF")
	branchField := fieldBox(branchView, !focusBase)
	baseField := fieldBox(baseView, focusBase)

	preview := styleDimmer.Render(pathPreview) + "\n" + styleDimmer.Render(sessionPreview)

	primaryStyle := styleButtonPrimary
	if hover == hitModalPrimary {
		primaryStyle = primaryStyle.Background(colorHoverAccentBg)
	}
	secondaryStyle := styleButtonGhost
	if hover == hitModalSecondary {
		secondaryStyle = secondaryStyle.Background(colorHoverBg)
	}
	const primaryText, secondaryText = "enter — create", "esc — cancel"
	actions := primaryStyle.Render(primaryText) + "    " + secondaryStyle.Render(secondaryText) +
		"    " + styleDimmer.Render("tab — switch field")

	// prefix's line count locates exactly where actions lands, without
	// hand-counting how many lines branchField/baseField/preview span.
	prefix := lipgloss.NewStyle().Foreground(colorText).Render(title) + "\n" +
		styleDimmer.Render(sub) + "\n\n" +
		branchLabel + "\n" + branchField + "\n\n" +
		baseLabel + "\n" + baseField + "\n\n" +
		preview + "\n\n"
	actionsLine := strings.Count(prefix, "\n")

	modal := styleModal.Width(56).Render(prefix + actions)
	ox, oy := modalContentOrigin()
	y := oy + actionsLine
	regions := []hitRegion{
		{x0: ox, y0: y, x1: ox + lipgloss.Width(primaryText), y1: y + 1, kind: hitModalPrimary},
		{x0: ox + lipgloss.Width(primaryText) + 4, y0: y, x1: ox + lipgloss.Width(primaryText) + 4 + lipgloss.Width(secondaryText), y1: y + 1, kind: hitModalSecondary},
	}
	return modal, regions
}

func launchNote(launch bool) string {
	if launch {
		return "session starts immediately"
	}
	return "no session until you attach"
}

func fieldBox(value string, focused bool) string {
	border := colorFaint
	if focused {
		border = colorAccent
	}
	s := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(border).Padding(0, 1).Width(48)
	return s.Render(value)
}

// renderConfirmModal draws the generalized confirm dialog, used for both
// merge and remove.
func renderConfirmModal(title string, steps []string, hover hitKind) (string, []hitRegion) {
	primaryStyle := styleButtonPrimary
	if hover == hitModalPrimary {
		primaryStyle = primaryStyle.Background(colorHoverAccentBg)
	}
	secondaryStyle := styleButtonGhost
	if hover == hitModalSecondary {
		secondaryStyle = secondaryStyle.Background(colorHoverBg)
	}
	const primaryText, secondaryText = "y — confirm", "n — cancel"
	actions := primaryStyle.Render(primaryText) + "    " + secondaryStyle.Render(secondaryText)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(colorText).Render(title))
	b.WriteString("\n\n")
	for _, s := range steps {
		b.WriteString(styleDimmer.Render(s))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	actionsLine := strings.Count(b.String(), "\n")
	b.WriteString(actions)

	modal := styleModal.Width(50).Render(b.String())
	ox, oy := modalContentOrigin()
	y := oy + actionsLine
	regions := []hitRegion{
		{x0: ox, y0: y, x1: ox + lipgloss.Width(primaryText), y1: y + 1, kind: hitModalPrimary},
		{x0: ox + lipgloss.Width(primaryText) + 4, y0: y, x1: ox + lipgloss.Width(primaryText) + 4 + lipgloss.Width(secondaryText), y1: y + 1, kind: hitModalSecondary},
	}
	return modal, regions
}

// renderPaletteModal draws the ':' command palette: a query input and the
// filtered command list, with the keyboard-selected row and (independently)
// the mouse-hovered row both marked, matching the design's separate `c.sel`
// and `style-hover` treatments.
func renderPaletteModal(queryView string, commands [][2]string, idx, hoverIdx int) (string, []hitRegion) {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Render(": ") + queryView)
	b.WriteString("\n\n")
	firstRowLine := strings.Count(b.String(), "\n")
	if len(commands) == 0 {
		b.WriteString(styleDimmer.Render("no matching commands"))
	}
	var regions []hitRegion
	ox, oy := modalContentOrigin()
	for i, c := range commands {
		label := c[0]
		key := c[1]
		line := label
		gap := max(40-len(label)-len(key), 1)
		line += strings.Repeat(" ", gap) + key
		switch {
		case i == idx:
			b.WriteString(lipgloss.NewStyle().Background(colorSelectedBg).Foreground(colorText).Render(padVisible(line, 48)))
		case i == hoverIdx:
			b.WriteString(lipgloss.NewStyle().Background(colorHoverBg).Foreground(colorText).Render(padVisible(line, 48)))
		default:
			b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(line))
		}
		b.WriteString("\n")
		y := oy + firstRowLine + i
		regions = append(regions, hitRegion{x0: ox, y0: y, x1: ox + 48, y1: y + 1, kind: hitPaletteRow, idx: i})
	}
	modal := styleModal.Width(52).Render(strings.TrimRight(b.String(), "\n"))
	return modal, regions
}

// renderHelpModal draws the '?' keybinds + settings overlay. It has no
// buttons of its own — clicking anywhere outside it (the backdrop) closes
// it, same as esc.
func renderHelpModal(settings [][2]string) string {
	var left, right strings.Builder
	half := (len(keybindHelp) + 1) / 2
	for i, k := range keybindHelp {
		line := lipgloss.NewStyle().Foreground(colorAccentLight).Render(padVisible(k[0], 8)) +
			lipgloss.NewStyle().Foreground(colorDim).Render(k[1]) + "\n"
		if i < half {
			left.WriteString(line)
		} else {
			right.WriteString(line)
		}
	}
	binds := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(28).Render(strings.TrimRight(left.String(), "\n")),
		lipgloss.NewStyle().Width(28).Render(strings.TrimRight(right.String(), "\n")))

	var sb strings.Builder
	for _, s := range settings {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(padVisible(s[0], 20)) +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#cfd3e5")).Render(s[1]) + "\n")
	}

	title := lipgloss.NewStyle().Foreground(colorText).Render("keybinds & settings") + "  " +
		styleDimmer.Render("~/.config/wtm/config.yaml")

	body := title + "\n\n" + binds + "\n\n" + strings.Repeat("─", 56) + "\n\n" + strings.TrimRight(sb.String(), "\n")
	return styleModal.Width(60).Render(body)
}
