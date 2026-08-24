package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/gitops"
)

const diffFileListWidth = 30

// renderDiffTab draws the two-pane diff review: a file list on the left
// (numstat +/- counts) and the selected file's unified diff hunks on the
// right, with a merge/discard action row along the bottom.
func renderDiffTab(wt Worktree, files []gitops.FileDiff, fileIdx int, lines []gitops.DiffLine, loading bool, err error, width, height int) string {
	summary := diffSummary(files)
	summaryLine := styleKicker.Render(summary)

	fileListW := diffFileListWidth
	if fileListW > width/3 {
		fileListW = width / 3
	}
	hunkW := width - fileListW - 3

	maxRows := max(height-1, 1)
	var fileRows []string
	fileRows = append(fileRows, summaryLine, "")
	for i, f := range files {
		if len(fileRows) >= maxRows {
			fileRows = append(fileRows, styleDimmer.Render(fmt.Sprintf("… %d more", len(files)-i)))
			break
		}
		row := clipLeft(f.Path, max(fileListW-9, 1)) +
			" " + lipgloss.NewStyle().Foreground(colorAccentLight).Render(fmt.Sprintf("+%d", f.Adds)) +
			" " + styleDimmer.Render(fmt.Sprintf("-%d", f.Dels))
		if i == fileIdx {
			row = lipgloss.NewStyle().Background(lipgloss.Color("#242038")).Render(padVisible(row, fileListW))
		}
		fileRows = append(fileRows, row)
	}
	fileList := lipgloss.NewStyle().Width(fileListW).Render(strings.Join(fileRows, "\n"))

	var hunkBody string
	switch {
	case err != nil:
		hunkBody = styleError.Render("diff error: " + err.Error())
	case loading:
		hunkBody = styleDimmer.Render("loading diff…")
	case len(files) == 0:
		hunkBody = styleDimmer.Render("no changes relative to " + wt.Base)
	default:
		hunkLines := make([]string, 0, len(lines))
		for _, l := range lines {
			hunkLines = append(hunkLines, renderDiffLine(l, hunkW))
		}
		// Diffs routinely run longer than the pane — no scroll state yet,
		// so clip rather than let the layout blow past the terminal.
		if limit := max(height-1, 1); len(hunkLines) > limit {
			hunkLines = hunkLines[:limit-1]
			hunkLines = append(hunkLines, styleDimmer.Render("… truncated — press ] for the next file"))
		}
		hunkBody = strings.Join(hunkLines, "\n")
	}
	hunkPane := lipgloss.NewStyle().Width(hunkW).Render(hunkBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, fileList, " │ ", hunkPane)

	mergeHint := lipgloss.NewStyle().Foreground(colorAccentLight).Render(fmt.Sprintf("m — merge into %s", wt.Base))
	discardHint := styleDimmer.Render("x — discard worktree")
	footer := mergeHint + "    " + discardHint

	return body + "\n" + strings.Repeat("─", width) + "\n" + footer
}

func diffSummary(files []gitops.FileDiff) string {
	if len(files) == 0 {
		return "no changes"
	}
	adds, dels := 0, 0
	for _, f := range files {
		adds += f.Adds
		dels += f.Dels
	}
	return fmt.Sprintf("%d files  +%d −%d", len(files), adds, dels)
}

func renderDiffLine(l gitops.DiffLine, width int) string {
	switch l.Kind {
	case gitops.DiffHunkHeader:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#b5abfc")).Render(clipPad(l.Text, width))
	case gitops.DiffAdd:
		bg := lipgloss.NewStyle().Background(lipgloss.Color("#241f38")).Foreground(colorAdd)
		return bg.Render(fmt.Sprintf("%4d ", l.LineNo)) + bg.Render(clipPad("+"+l.Text, max(width-5, 1)))
	case gitops.DiffDel:
		bg := lipgloss.NewStyle().Foreground(colorDel)
		return bg.Render(fmt.Sprintf("%4d ", l.LineNo)) + bg.Render(clipPad("-"+l.Text, max(width-5, 1)))
	default:
		fg := lipgloss.NewStyle().Foreground(lipgloss.Color("#b2b6ca"))
		return styleDimmer.Render(fmt.Sprintf("%4d ", l.LineNo)) + fg.Render(clipPad(" "+l.Text, max(width-5, 1)))
	}
}
