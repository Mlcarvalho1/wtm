package termpty

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/vt"
)

// fKeys pairs bubbletea's F1-F20 key types with vt's equivalent key codes, in
// matching order, so toVTKey can zip them into a lookup table below instead
// of writing out twenty near-identical switch cases by hand.
var fKeys = []struct {
	tea tea.KeyType
	vt  rune
}{
	{tea.KeyF1, vt.KeyF1}, {tea.KeyF2, vt.KeyF2}, {tea.KeyF3, vt.KeyF3}, {tea.KeyF4, vt.KeyF4},
	{tea.KeyF5, vt.KeyF5}, {tea.KeyF6, vt.KeyF6}, {tea.KeyF7, vt.KeyF7}, {tea.KeyF8, vt.KeyF8},
	{tea.KeyF9, vt.KeyF9}, {tea.KeyF10, vt.KeyF10}, {tea.KeyF11, vt.KeyF11}, {tea.KeyF12, vt.KeyF12},
	{tea.KeyF13, vt.KeyF13}, {tea.KeyF14, vt.KeyF14}, {tea.KeyF15, vt.KeyF15}, {tea.KeyF16, vt.KeyF16},
	{tea.KeyF17, vt.KeyF17}, {tea.KeyF18, vt.KeyF18}, {tea.KeyF19, vt.KeyF19}, {tea.KeyF20, vt.KeyF20},
}

// ctrlLetters pairs bubbletea's KeyCtrlA-Z with the lowercase letter vt
// expects as a ctrl-modified Code (vt.Emulator.SendKey encodes the actual
// control byte from Code+ModCtrl itself).
//
// KeyCtrlI and KeyCtrlM are deliberately absent: bubbletea gives them the
// exact same numeric KeyType as KeyTab and KeyEnter respectively (both
// pairs decode from the same raw byte, 0x09/0x0D — there's no way to tell
// "Tab" from "Ctrl+I" at this layer, they're the same keystroke), so
// including them here would shadow plainKeys below and turn every plain
// Tab/Enter press into a ctrl-chord.
var ctrlLetters = map[tea.KeyType]rune{
	tea.KeyCtrlA: 'a', tea.KeyCtrlB: 'b', tea.KeyCtrlC: 'c', tea.KeyCtrlD: 'd',
	tea.KeyCtrlE: 'e', tea.KeyCtrlF: 'f', tea.KeyCtrlG: 'g', tea.KeyCtrlH: 'h',
	tea.KeyCtrlJ: 'j', tea.KeyCtrlK: 'k', tea.KeyCtrlL: 'l',
	tea.KeyCtrlN: 'n', tea.KeyCtrlO: 'o', tea.KeyCtrlP: 'p',
	tea.KeyCtrlQ: 'q', tea.KeyCtrlR: 'r', tea.KeyCtrlS: 's', tea.KeyCtrlT: 't',
	tea.KeyCtrlU: 'u', tea.KeyCtrlV: 'v', tea.KeyCtrlW: 'w', tea.KeyCtrlX: 'x',
	tea.KeyCtrlY: 'y', tea.KeyCtrlZ: 'z',
}

// plainKeys pairs bubbletea key types with vt key codes for keys that carry
// no modifier information of their own.
var plainKeys = map[tea.KeyType]rune{
	tea.KeyEnter:     vt.KeyEnter,
	tea.KeyTab:       vt.KeyTab,
	tea.KeyEsc:       vt.KeyEscape,
	tea.KeyBackspace: vt.KeyBackspace,
	tea.KeyDelete:    vt.KeyDelete,
	tea.KeyInsert:    vt.KeyInsert,
	tea.KeySpace:     vt.KeySpace,
	tea.KeyUp:        vt.KeyUp,
	tea.KeyDown:      vt.KeyDown,
	tea.KeyLeft:      vt.KeyLeft,
	tea.KeyRight:     vt.KeyRight,
	tea.KeyHome:      vt.KeyHome,
	tea.KeyEnd:       vt.KeyEnd,
	tea.KeyPgUp:      vt.KeyPgUp,
	tea.KeyPgDown:    vt.KeyPgDown,
}

// ctrlArrows pairs bubbletea's ctrl+arrow key types with the plain vt arrow
// code — the ctrl modifier itself is added by the caller.
var ctrlArrows = map[tea.KeyType]rune{
	tea.KeyCtrlUp:   vt.KeyUp,
	tea.KeyCtrlDown: vt.KeyDown,
}

// toVTKey converts a bubbletea key event into the vt.KeyPressEvent that
// Emulator.SendKey expects, so a forwarded keystroke is encoded the way the
// attached terminal's *current* modes (application cursor keys, etc.)
// require rather than as a hardcoded escape sequence.
func toVTKey(msg tea.KeyMsg) vt.KeyPressEvent {
	mod := vt.KeyMod(0)
	if msg.Alt {
		mod |= vt.ModAlt
	}

	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		r := msg.Runes[0]
		return vt.KeyPressEvent{Code: r, Text: string(msg.Runes), Mod: mod}
	}
	if msg.Type == tea.KeyShiftTab {
		return vt.KeyPressEvent{Code: vt.KeyTab, Mod: mod | vt.ModShift}
	}
	if r, ok := ctrlLetters[msg.Type]; ok {
		return vt.KeyPressEvent{Code: r, Mod: mod | vt.ModCtrl}
	}
	if r, ok := ctrlArrows[msg.Type]; ok {
		return vt.KeyPressEvent{Code: r, Mod: mod | vt.ModCtrl}
	}
	if r, ok := plainKeys[msg.Type]; ok {
		return vt.KeyPressEvent{Code: r, Mod: mod}
	}
	for _, fk := range fKeys {
		if msg.Type == fk.tea {
			return vt.KeyPressEvent{Code: fk.vt, Mod: mod}
		}
	}

	// Fallback: whatever bubbletea decoded the rune as, best-effort.
	if len(msg.Runes) > 0 {
		return vt.KeyPressEvent{Code: msg.Runes[0], Text: string(msg.Runes), Mod: mod}
	}
	return vt.KeyPressEvent{Mod: mod}
}
