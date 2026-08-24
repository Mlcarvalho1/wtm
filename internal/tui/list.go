package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
)

func newTable() table.Model {
	columns := []table.Column{
		{Title: "", Width: 2},
		{Title: "Repo", Width: 14},
		{Title: "Branch", Width: 22},
		{Title: "Status", Width: 10},
		{Title: "Session", Width: 26},
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)
	t.SetStyles(tableStyles())
	return t
}

func rowsFromWorktrees(wts []Worktree) []table.Row {
	rows := make([]table.Row, 0, len(wts))
	for _, w := range wts {
		rows = append(rows, table.Row{
			w.State.Icon(),
			w.RepoName,
			w.Branch,
			statusCell(w),
			w.State.String(),
		})
	}
	return rows
}

func statusCell(w Worktree) string {
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
