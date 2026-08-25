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
// right, with a merge/discard action row along the bottom. Both the file
// list and the hunk pane are clickable/hoverable, and the hunk pane scrolls
// (scroll is a line offset, clamped internally to what the content and
// height actually allow — callers just track a monotonically-adjusted
// counter without needing to know the max).
func renderDiffTab(wt Worktree, files []gitops.FileDiff, fileIdx int, lines []gitops.DiffLine, loading bool, err error, width, height, scroll int, rc renderCtx) string {
	summary := diffSummary(files)
	summaryLine := styleKicker.Render(summary)

	fileListW := diffFileListWidth
	if fileListW > width/3 {
		fileListW = width / 3
	}
	hunkW := width - fileListW - 3
	bodyH := max(height-2, 1) // reserves the separator + footer rows below

	maxRows := bodyH
	var fileRows []string
	fileRows = append(fileRows, summaryLine, "")
	fileListRC := rc
	for i, f := range files {
		if len(fileRows) >= maxRows {
			fileRows = append(fileRows, styleDimmer.Render(fmt.Sprintf("… %d more", len(files)-i)))
			break
		}
		rowY := len(fileRows)
		row := clipLeft(f.Path, max(fileListW-9, 1)) +
			" " + lipgloss.NewStyle().Foreground(colorAccentLight).Render(fmt.Sprintf("+%d", f.Adds)) +
			" " + styleDimmer.Render(fmt.Sprintf("-%d", f.Dels))
		switch {
		case i == fileIdx:
			row = lipgloss.NewStyle().Background(colorSelectedBg).Render(padVisible(row, fileListW))
		case fileListRC.hovered(hitDiffFile, i):
			row = lipgloss.NewStyle().Background(colorHoverBg).Render(padVisible(row, fileListW))
		}
		fileListRC.addHit(hitDiffFile, i, 0, rowY, fileListW, rowY+1)
		fileRows = append(fileRows, row)
	}
	fileList := lipgloss.NewStyle().Width(fileListW).Render(strings.Join(fileRows, "\n"))

	visibleHunkRows := bodyH
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
		maxScroll := max(len(hunkLines)-visibleHunkRows, 0)
		scroll = min(max(scroll, 0), maxScroll)
		visible := hunkLines[scroll:min(scroll+visibleHunkRows, len(hunkLines))]
		hunkBody = strings.Join(visible, "\n")
	}
	hunkPane := lipgloss.NewStyle().Width(hunkW).Render(hunkBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, fileList, " │ ", hunkPane)
	// Pinned to bottom, matching the design's flex layout: without this, a
	// short file list or diff would let the separator+footer float right
	// under the content instead of at the pane's actual bottom edge, which
	// also lets the merge/discard hit regions below assume a fixed row.
	body = fitHeight(body, width, bodyH)

	mergeStyle := lipgloss.NewStyle().Foreground(colorAccentLight)
	if rc.hovered(hitMergeButton, 0) {
		mergeStyle = mergeStyle.Background(colorHoverAccentBg)
	}
	mergeText := fmt.Sprintf("m — merge into %s", wt.Base)
	mergeHint := mergeStyle.Render(mergeText)

	discardStyle := lipgloss.NewStyle().Foreground(colorDim)
	if rc.hovered(hitDiscardButton, 0) {
		discardStyle = discardStyle.Background(colorHoverBg)
	}
	const discardText = "x — discard worktree"
	discardHint := discardStyle.Render(discardText)

	footerY := bodyH + 1
	rc.addHit(hitMergeButton, 0, 0, footerY, lipgloss.Width(mergeText), footerY+1)
	rc.addHit(hitDiscardButton, 0, lipgloss.Width(mergeText)+4, footerY, lipgloss.Width(mergeText)+4+lipgloss.Width(discardText), footerY+1)

	left := mergeHint + "    " + discardHint
	right := styleDimmer.Render(gitLong(wt))
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	footer := left + strings.Repeat(" ", gap) + right

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
		bg := lipgloss.NewStyle().Background(colorDiffHunkBg).Foreground(lipgloss.Color("#b5abfc"))
		return bg.Render(clipPad(l.Text, width))
	case gitops.DiffAdd:
		bg := lipgloss.NewStyle().Background(colorDiffAddBg).Foreground(colorAdd)
		return bg.Render(fmt.Sprintf("%4d ", l.LineNo)) + bg.Render(clipPad("+"+l.Text, max(width-5, 1)))
	case gitops.DiffDel:
		bg := lipgloss.NewStyle().Background(colorDiffDelBg).Foreground(colorDel)
		return bg.Render(fmt.Sprintf("%4d ", l.LineNo)) + bg.Render(clipPad("-"+l.Text, max(width-5, 1)))
	default:
		fg := lipgloss.NewStyle().Foreground(lipgloss.Color("#b2b6ca"))
		return styleDimmer.Render(fmt.Sprintf("%4d ", l.LineNo)) + fg.Render(clipPad(" "+l.Text, max(width-5, 1)))
	}
}
