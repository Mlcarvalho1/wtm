package termpty

import (
	tea "github.com/charmbracelet/bubbletea"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

// mouseButtons pairs bubbletea's mouse button constants with vt's — see
// toVTKey's own doc comment in keys.go for why this codebase always
// translates through an explicit table rather than assuming the two
// libraries' numeric values line up.
var mouseButtons = map[tea.MouseButton]vt.MouseButton{
	tea.MouseButtonLeft:       vt.MouseLeft,
	tea.MouseButtonMiddle:     vt.MouseMiddle,
	tea.MouseButtonRight:      vt.MouseRight,
	tea.MouseButtonWheelUp:    vt.MouseWheelUp,
	tea.MouseButtonWheelDown:  vt.MouseWheelDown,
	tea.MouseButtonWheelLeft:  vt.MouseWheelLeft,
	tea.MouseButtonWheelRight: vt.MouseWheelRight,
	tea.MouseButtonBackward:   vt.MouseBackward,
	tea.MouseButtonForward:    vt.MouseForward,
}

// toVTMouse converts a bubbletea mouse event into the uv.Mouse value vt's
// concrete click/release/wheel/motion event types wrap (vt.Mouse itself is
// only the uv.MouseEvent interface those types satisfy, not this struct).
// msg.X/Y are expected already translated into the emulator's own cell grid
// (see Attachment.ForwardMouse) — this only handles button/modifier
// encoding.
func toVTMouse(msg tea.MouseMsg) uv.Mouse {
	var mod vt.KeyMod
	if msg.Shift {
		mod |= vt.ModShift
	}
	if msg.Alt {
		mod |= vt.ModAlt
	}
	if msg.Ctrl {
		mod |= vt.ModCtrl
	}
	btn, ok := mouseButtons[msg.Button]
	if !ok {
		btn = vt.MouseNone
	}
	return uv.Mouse{X: msg.X, Y: msg.Y, Button: btn, Mod: mod}
}
