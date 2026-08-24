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

// renderNewWorktreeModal draws the combined branch+base-ref form (tab moves
// between fields; both are visible at once, unlike the CLI's old two-step
// sequential prompts).
func renderNewWorktreeModal(repoName string, launch bool, branchView, baseView string, focusBase bool, pathPreview, sessionPreview string) string {
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

	actions := styleButtonPrimary.Render("enter — create") + "    " + styleButtonGhost.Render("esc — cancel") +
		"    " + styleDimmer.Render("tab — switch field")

	body := lipgloss.NewStyle().Foreground(colorText).Render(title) + "\n" +
		styleDimmer.Render(sub) + "\n\n" +
		branchLabel + "\n" + branchField + "\n\n" +
		baseLabel + "\n" + baseField + "\n\n" +
		preview + "\n\n" +
		actions

	return styleModal.Width(56).Render(body)
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
func renderConfirmModal(title string, steps []string) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(colorText).Render(title))
	b.WriteString("\n\n")
	for _, s := range steps {
		b.WriteString(styleDimmer.Render(s))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleButtonPrimary.Render("y — confirm") + "    " + styleButtonGhost.Render("n — cancel"))
	return styleModal.Width(50).Render(b.String())
}

// renderPaletteModal draws the ':' command palette: a query input and the
// filtered command list, with the highlighted row marked.
func renderPaletteModal(queryView string, commands [][2]string, idx int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Render(": ") + queryView)
	b.WriteString("\n\n")
	if len(commands) == 0 {
		b.WriteString(styleDimmer.Render("no matching commands"))
	}
	for i, c := range commands {
		label := c[0]
		key := c[1]
		line := label
		gap := max(40-len(label)-len(key), 1)
		line += strings.Repeat(" ", gap) + key
		if i == idx {
			b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#242038")).Foreground(colorText).Render(padVisible(line, 48)))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(line))
		}
		b.WriteString("\n")
	}
	return styleModal.Width(52).Render(strings.TrimRight(b.String(), "\n"))
}

// renderHelpModal draws the '?' keybinds + settings overlay.
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
