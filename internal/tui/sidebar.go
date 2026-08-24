package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/watch"
)

// gitCell is the compact "*↑2↓1" / "clean" status cell shown next to a
// branch name in the sidebar.
func gitCell(w Worktree) string {
	if !w.Dirty && w.Ahead == 0 && w.Behind == 0 {
		return "clean"
	}
	s := ""
	if w.Dirty {
		s += "*"
	}
	if w.Ahead > 0 {
		s += fmt.Sprintf("↑%d", w.Ahead)
	}
	if w.Behind > 0 {
		s += fmt.Sprintf("↓%d", w.Behind)
	}
	return s
}

// gitLong is the spelled-out version of gitCell, shown in the session and
// diff tab footers.
func gitLong(w Worktree) string {
	var parts []string
	if w.Dirty {
		parts = append(parts, "uncommitted changes")
	}
	if w.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d ahead", w.Ahead))
	}
	if w.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d behind", w.Behind))
	}
	if len(parts) == 0 {
		return "clean, in sync"
	}
	return strings.Join(parts, " · ")
}

func sessionLabel(w Worktree) string {
	if w.State == watch.StateStopped {
		return "no session"
	}
	return w.Session
}

// sidebarRow is one flattened line of the sidebar's content, before vertical
// windowing: either a repo group header or the first/second line of a
// worktree row.
type sidebarRow struct {
	text     string
	selected bool // true when this line belongs to the cursor's worktree
}

// renderSidebar draws the repo-grouped worktree list: a group header per
// repo, two lines per worktree row (branch+status, then state+session),
// scrolled so the cursor stays in view.
func renderSidebar(rows []Worktree, cursor int, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	var lines []sidebarRow
	selectedLine := 0

	var lastRepo string
	groupCount := 0
	// Count sizes up front so the header can show "N" per repo without a
	// second pass — worktrees are already grouped contiguously by repo,
	// since LoadWorktrees iterates cfg.Repos in order.
	counts := map[string]int{}
	for _, w := range rows {
		counts[w.RepoName]++
	}

	for i, w := range rows {
		if w.RepoName != lastRepo {
			lastRepo = w.RepoName
			groupCount = counts[w.RepoName]
			labelText := strings.ToUpper(w.RepoName)
			countText := fmt.Sprintf("%d", groupCount)
			ruleWidth := max(width-len(labelText)-len(countText)-2, 0)
			header := styleKicker.Render(labelText) + " " +
				styleDimmer.Render(strings.Repeat("─", ruleWidth)) + " " +
				styleDimmer.Render(countText)
			lines = append(lines, sidebarRow{text: padVisible(header, width)})
		}

		isSel := i == cursor
		if isSel {
			selectedLine = len(lines)
		}

		dotStyle := lipgloss.NewStyle().Foreground(stateColor(w.State))
		branchColor := lipgloss.NewStyle().Foreground(colorText)
		gitColor := styleDimmer

		indicator := dot(w.State != watch.StateStopped)
		if w.State == watch.StateNeedsInput {
			dotStyle = dotStyle.Blink(true)
		}

		branchW := width - 2 /*dot+space*/ - 1 /*space*/ - lipgloss.Width(gitCell(w))
		line1 := dotStyle.Render(indicator) + " " + branchColor.Render(clipPad(w.Branch, max(branchW, 1))) + " " + gitColor.Render(gitCell(w))
		line1 = padVisible(line1, width)

		stColor := lipgloss.NewStyle().Foreground(stateColor(w.State))
		st := stColor.Render(stateLabel(w.State))
		sess := styleDimmer.Render(clipPad(sessionLabel(w), max(width-lipgloss.Width(stateLabel(w.State))-5, 1)))
		line2 := "  " + st + styleDimmer.Render(" · ") + sess
		line2 = padVisible(line2, width)

		lines = append(lines, sidebarRow{text: line1, selected: isSel})
		lines = append(lines, sidebarRow{text: line2, selected: isSel})
	}

	// Window the flattened lines so the cursor's rows stay visible.
	top := 0
	if len(lines) > height {
		top = selectedLine - height/2
		if top < 0 {
			top = 0
		}
		if top > len(lines)-height {
			top = len(lines) - height
		}
	}
	bottom := min(top+height, len(lines))

	out := make([]string, 0, height)
	for i := top; i < bottom; i++ {
		l := lines[i]
		if l.selected {
			out = append(out, lipgloss.NewStyle().Background(lipgloss.Color("#242038")).Render(l.text))
		} else {
			out = append(out, l.text)
		}
	}
	blank := strings.Repeat(" ", width)
	for len(out) < height {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}

// padVisible pads s with spaces so its rendered (ANSI-stripped) width
// reaches width, without disturbing any styling already applied to s.
func padVisible(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
