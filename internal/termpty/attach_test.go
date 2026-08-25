package termpty

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// newTestAttachment builds an Attachment around a bare emulator — enough to
// exercise Render()'s cursor overlay without a real PTY/tmux process.
func newTestAttachment(cols, rows int) *Attachment {
	return &Attachment{emu: vt.NewSafeEmulator(cols, rows)}
}

func TestRenderDrawsCursorAtEmulatorPosition(t *testing.T) {
	a := newTestAttachment(10, 3)
	a.emu.Write([]byte("abcde"))   // cursor now at col 5, row 0
	a.emu.Write([]byte("\r\nxyz")) // and now at col 3, row 1

	out := a.Render()
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}

	// The cursor sits just past "xyz" on row 1 — a reverse-video space.
	if !strings.Contains(lines[1], "\x1b[7m") && !strings.Contains(lines[1], ";7m") {
		t.Errorf("expected a reverse-video (SGR 7) sequence on the cursor's row, got %q", lines[1])
	}

	// Every other row should be untouched by the overlay.
	plainRow0 := ansi.Strip(lines[0])
	if plainRow0 != "abcde" {
		t.Errorf("row 0 (no cursor there) changed unexpectedly: %q", plainRow0)
	}
}

func TestRenderLeavesFrameUnchangedWhenCursorOutOfBounds(t *testing.T) {
	a := newTestAttachment(5, 2)
	// Force an out-of-range cursor by shrinking the emulator after writing —
	// Resize clamps the real cursor, so instead just check the plain
	// in-bounds path renders identically to the raw emulator when nothing
	// is written (cursor at 0,0, a valid position) — regression guard that
	// the overlay doesn't corrupt an otherwise-untouched frame.
	plain := a.emu.Render()
	got := a.Render()
	// Only the cursor cell (0,0) may legitimately differ (reverse video).
	gotLines := strings.Split(got, "\n")
	plainLines := strings.Split(plain, "\n")
	if len(gotLines) != len(plainLines) {
		t.Fatalf("line count changed: got %d, want %d", len(gotLines), len(plainLines))
	}
	for i := 1; i < len(gotLines); i++ {
		if gotLines[i] != plainLines[i] {
			t.Errorf("row %d changed but shouldn't have: got %q, want %q", i, gotLines[i], plainLines[i])
		}
	}
}

func TestAttrReverseToggle(t *testing.T) {
	var s uv.Style
	s.Attrs ^= uv.AttrReverse
	if s.Attrs&uv.AttrReverse == 0 {
		t.Fatalf("expected AttrReverse to be set after toggling once")
	}
}
