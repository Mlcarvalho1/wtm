package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Mlcarvalho1/wtm/internal/gitops"
)

const diffFileListWidth = 30

// renderDiffTab draws the two-pane diff review: a file list on the left
// (numstat +/- counts) and the selected file's diff hunks on the right —
// either unified or, when split is true, a GitHub/VSCode-style side-by-side
// layout (see renderSplitRows) — with a merge/discard/split-toggle action
// row along the bottom. Both the file list and the hunk pane are
// clickable/hoverable, and the hunk pane scrolls (scroll is a line offset,
// clamped internally to what the content and height actually allow —
// callers just track a monotonically-adjusted counter without needing to
// know the max).
func renderDiffTab(wt Worktree, files []gitops.FileDiff, fileIdx int, lines []gitops.DiffLine, loading bool, err error, width, height, scroll int, split bool, rc renderCtx) string {
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
		hunkLines := buildHunkRows(lines, hunkW, split)
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

	splitStyle := lipgloss.NewStyle().Foreground(colorDim)
	if rc.hovered(hitSplitToggle, 0) {
		splitStyle = splitStyle.Background(colorHoverBg)
	}
	splitText := "s — split diff"
	if split {
		splitText = "s — unified diff"
	}
	splitHint := splitStyle.Render(splitText)

	footerY := bodyH + 1
	discardX0 := lipgloss.Width(mergeText) + 4
	splitX0 := discardX0 + lipgloss.Width(discardText) + 4
	rc.addHit(hitMergeButton, 0, 0, footerY, lipgloss.Width(mergeText), footerY+1)
	rc.addHit(hitDiscardButton, 0, discardX0, footerY, discardX0+lipgloss.Width(discardText), footerY+1)
	rc.addHit(hitSplitToggle, 0, splitX0, footerY, splitX0+lipgloss.Width(splitText), footerY+1)

	left := mergeHint + "    " + discardHint + "    " + splitHint
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
	if l.Kind == gitops.DiffHunkHeader {
		bg := lipgloss.NewStyle().Background(colorDiffHunkBg).Foreground(lipgloss.Color("#b5abfc"))
		return bg.Render(clipPad(l.Text, width))
	}
	return renderDiffCell(&l, width)
}

// renderDiffCell renders one add/del/context line into a width-wide cell —
// the unified view's whole row, or one side of a split-view row. l is nil
// for the unpaired half of a split row whose other side ran longer (a
// removal or addition run of unequal length within one hunk).
func renderDiffCell(l *gitops.DiffLine, width int) string {
	if l == nil {
		return strings.Repeat(" ", max(width, 0))
	}
	switch l.Kind {
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

// buildHunkRows renders a file's diff lines into screen rows: one row per
// DiffLine for the unified view, or GitHub/VSCode-style paired columns (see
// renderSplitRows) when split is true.
func buildHunkRows(lines []gitops.DiffLine, width int, split bool) []string {
	if !split {
		rows := make([]string, 0, len(lines))
		for _, l := range lines {
			rows = append(rows, renderDiffLine(l, width))
		}
		return rows
	}
	return renderSplitRows(lines, width)
}

// renderSplitRows lays a file's diff lines out as a side-by-side (old |
// new) split view: a hunk header spans the full width, a context line
// repeats identically on both sides, and each hunk's contiguous run of
// removals is paired row-by-row against its immediately following run of
// additions — unified diff always orders a changed block that way (all
// its removals, then all its additions), so pairing consecutive runs lines
// up before/after edits to the same lines the way GitHub and VSCode's split
// diff do. A run longer than its counterpart leaves a blank cell on the
// shorter side for its extra lines.
func renderSplitRows(lines []gitops.DiffLine, width int) []string {
	leftW := (width - 1) / 2
	rightW := width - 1 - leftW
	sep := styleDimmer.Render("│")

	var rows []string
	i := 0
	for i < len(lines) {
		switch lines[i].Kind {
		case gitops.DiffHunkHeader:
			rows = append(rows, renderDiffLine(lines[i], width))
			i++

		case gitops.DiffContext:
			l := lines[i]
			rows = append(rows, renderDiffCell(&l, leftW)+sep+renderDiffCell(&l, rightW))
			i++

		case gitops.DiffAdd:
			// A pure-addition block: no removals immediately preceded it
			// (e.g. lines added at the very start of a hunk).
			j := i
			for j < len(lines) && lines[j].Kind == gitops.DiffAdd {
				j++
			}
			for k := i; k < j; k++ {
				rows = append(rows, renderDiffCell(nil, leftW)+sep+renderDiffCell(&lines[k], rightW))
			}
			i = j

		default: // gitops.DiffDel: a removal run, and whatever addition run follows it
			delStart := i
			for i < len(lines) && lines[i].Kind == gitops.DiffDel {
				i++
			}
			addStart := i
			for i < len(lines) && lines[i].Kind == gitops.DiffAdd {
				i++
			}
			dels, adds := lines[delStart:addStart], lines[addStart:i]
			for k := range max(len(dels), len(adds)) {
				var l, r *gitops.DiffLine
				if k < len(dels) {
					l = &dels[k]
				}
				if k < len(adds) {
					r = &adds[k]
				}
				rows = append(rows, renderDiffCell(l, leftW)+sep+renderDiffCell(r, rightW))
			}
		}
	}
	return rows
}
