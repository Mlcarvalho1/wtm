package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleError  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	stylePrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	styleHelp   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("15")).BorderBottom(true).BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("0")).Background(lipgloss.Color("212")).Bold(true)
	return s
}
