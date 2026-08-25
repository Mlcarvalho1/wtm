package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/watch"
)

// sessionHeadLine renders the one-line branch/state-badge/session-name
// header shared by the read-only session preview and the live attached view,
// so the two stay visually consistent.
func sessionHeadLine(wt Worktree) string {
	// A single-line "pill" — no lipgloss Border here, which (even with
	// Padding(0,1), a purely horizontal setting) still draws a top and
	// bottom edge by default, silently making this "one line" badge three
	// lines tall and throwing off every fixed-height calculation below it.
	badge := lipgloss.NewStyle().
		Foreground(colorAccentLight).
		Render("[ " + stateLabel(wt.State) + " ]")

	return lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(wt.Branch) +
		"  " + badge + "  " + styleDimmer.Render(sessionLabel(wt))
}

// renderSessionTab draws the pane-preview tab: header line (branch, state
// badge, session name), the tailed capture-pane box, and worktree/session
// info boxes below it.
func renderSessionTab(wt Worktree, lastChanged time.Time, width, height int) string {
	head := sessionHeadLine(wt)

	paneHeaderRight := "no session"
	if wt.State != watch.StateStopped {
		if lastChanged.IsZero() {
			paneHeaderRight = "watching"
		} else {
			paneHeaderRight = "updated " + humanizeAgo(time.Since(lastChanged))
		}
	}
	paneHeader := styleDimmer.Render("capture-pane · tail")
	gap := max(width-4-lipgloss.Width(paneHeader)-lipgloss.Width(paneHeaderRight), 1)
	paneHeaderLine := paneHeader + strings.Repeat(" ", gap) + styleDimmer.Render(paneHeaderRight)

	paneInnerW := max(width-4, 1)
	// Everything else in this tab besides the pane's own lines: head(1) +
	// blank(1) + pane box's header+top+bottom border(3) + info-box
	// row(5, its own border+kicker+2 lines) = 10 fixed lines.
	paneInnerH := max(height-10, 3)
	var paneLines []string
	if wt.LastPane == "" {
		paneLines = []string{styleDimmer.Render("(no session — enter launches claude here)")}
	} else {
		raw := tailLines(strings.Split(strings.TrimRight(wt.LastPane, "\n"), "\n"), paneInnerH)
		for _, l := range raw {
			paneLines = append(paneLines, clipPad(l, paneInnerW))
		}
	}
	for len(paneLines) < paneInnerH {
		paneLines = append(paneLines, "")
	}

	// styleBox already carries a border + horizontal padding that together
	// consume 4 columns; paneLines above are pre-clipped to width-4, so the
	// box is left to size itself from that content rather than being told
	// a width too (Style.Width on a bordered style measures differently
	// across lipgloss versions — safer to just not fight it).
	paneBox := styleBox.Render(paneHeaderLine + "\n" + strings.Join(paneLines, "\n"))

	boxWidth := (width - 3) / 2
	worktreeBox := infoBox("WORKTREE", boxWidth, []string{
		wt.Path,
		fmt.Sprintf("base %s · %s", wt.Base, gitLong(wt)),
	})
	sessionBox := infoBox("SESSION", width-2-boxWidth-1, []string{
		fmt.Sprintf("%s%s", strings.ToUpper(stateLabel(wt.State)[:1]), stateLabel(wt.State)[1:]),
		sessionActivityLine(wt, lastChanged),
	})

	infoRow := lipgloss.JoinHorizontal(lipgloss.Top, worktreeBox, " ", sessionBox)

	return head + "\n\n" + paneBox + "\n" + infoRow
}

func sessionActivityLine(wt Worktree, lastChanged time.Time) string {
	if wt.State == watch.StateStopped {
		return "no session"
	}
	if lastChanged.IsZero() {
		return "watching for changes"
	}
	return "last changed " + humanizeAgo(time.Since(lastChanged)) + " ago"
}

func infoBox(title string, width int, lines []string) string {
	body := styleKicker.Render(title) + "\n"
	rendered := make([]string, len(lines))
	for i, l := range lines {
		style := lipgloss.NewStyle().Foreground(colorDim)
		if i == 0 {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#cfd3e5")) // neutral-300
		}
		rendered[i] = style.Render(clipPad(l, max(width-4, 1)))
	}
	body += strings.Join(rendered, "\n")
	return styleBox.Render(body) // see paneBox: content is pre-clipped to width-4, not re-Width()'d
}

// humanizeAgo renders a duration the way the design's compact timestamps
// read ("2s", "14m", "1h 12m").
func humanizeAgo(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", max(int(d.Seconds()), 0))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
