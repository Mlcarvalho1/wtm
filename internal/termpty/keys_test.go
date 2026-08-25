package termpty

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/vt"
)

func TestToVTKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want vt.KeyPressEvent
	}{
		{
			name: "plain rune",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
			want: vt.KeyPressEvent{Code: 'a', Text: "a"},
		},
		{
			name: "alt-modified rune",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a"), Alt: true},
			want: vt.KeyPressEvent{Code: 'a', Text: "a", Mod: vt.ModAlt},
		},
		{
			name: "enter",
			msg:  tea.KeyMsg{Type: tea.KeyEnter},
			want: vt.KeyPressEvent{Code: vt.KeyEnter},
		},
		{
			name: "escape",
			msg:  tea.KeyMsg{Type: tea.KeyEsc},
			want: vt.KeyPressEvent{Code: vt.KeyEscape},
		},
		{
			name: "backspace",
			msg:  tea.KeyMsg{Type: tea.KeyBackspace},
			want: vt.KeyPressEvent{Code: vt.KeyBackspace},
		},
		{
			name: "tab",
			msg:  tea.KeyMsg{Type: tea.KeyTab},
			want: vt.KeyPressEvent{Code: vt.KeyTab},
		},
		{
			name: "shift+tab",
			msg:  tea.KeyMsg{Type: tea.KeyShiftTab},
			want: vt.KeyPressEvent{Code: vt.KeyTab, Mod: vt.ModShift},
		},
		{
			name: "up arrow",
			msg:  tea.KeyMsg{Type: tea.KeyUp},
			want: vt.KeyPressEvent{Code: vt.KeyUp},
		},
		{
			name: "ctrl+up arrow",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlUp},
			want: vt.KeyPressEvent{Code: vt.KeyUp, Mod: vt.ModCtrl},
		},
		{
			name: "ctrl+c",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlC},
			want: vt.KeyPressEvent{Code: 'c', Mod: vt.ModCtrl},
		},
		{
			name: "ctrl+a",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlA},
			want: vt.KeyPressEvent{Code: 'a', Mod: vt.ModCtrl},
		},
		{
			name: "f5",
			msg:  tea.KeyMsg{Type: tea.KeyF5},
			want: vt.KeyPressEvent{Code: vt.KeyF5},
		},
		{
			name: "home",
			msg:  tea.KeyMsg{Type: tea.KeyHome},
			want: vt.KeyPressEvent{Code: vt.KeyHome},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toVTKey(tt.msg)
			if got != tt.want {
				t.Fatalf("toVTKey(%+v) = %+v, want %+v", tt.msg, got, tt.want)
			}
		})
	}
}
