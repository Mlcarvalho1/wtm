package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHitMapAtPicksMostRecentlyAdded(t *testing.T) {
	var h hitMap
	h.add(hitOverlayBackdrop, 0, 0, 0, 10, 10)
	h.add(hitModalPrimary, 0, 2, 2, 6, 3)

	if r, ok := h.at(3, 2); !ok || r.kind != hitModalPrimary {
		t.Fatalf("expected the button (registered after the backdrop) to win, got %+v ok=%v", r, ok)
	}
	if r, ok := h.at(0, 0); !ok || r.kind != hitOverlayBackdrop {
		t.Fatalf("expected the backdrop outside the button, got %+v ok=%v", r, ok)
	}
	if _, ok := h.at(20, 20); ok {
		t.Fatalf("expected no region outside both rectangles")
	}
}

func TestHitMapResetClears(t *testing.T) {
	var h hitMap
	h.add(hitSidebarRow, 0, 0, 0, 5, 1)
	h.reset()
	if _, ok := h.at(1, 0); ok {
		t.Fatalf("expected reset to clear previously added regions")
	}
}

func TestHitMapAddIgnoresEmptyRegions(t *testing.T) {
	var h hitMap
	h.add(hitSidebarRow, 0, 5, 5, 5, 5) // zero area
	h.add(hitSidebarRow, 0, 5, 5, 3, 8) // inverted
	if len(h.regions) != 0 {
		t.Fatalf("expected degenerate regions to be dropped, got %d", len(h.regions))
	}
}

func TestRenderCtxAddHitOffsetsByOrigin(t *testing.T) {
	h := &hitMap{}
	rc := renderCtx{hits: h, x0: 10, y0: 20}
	rc.addHit(hitDiffFile, 3, 1, 1, 4, 2)

	if r, ok := h.at(11, 21); !ok || r.kind != hitDiffFile || r.idx != 3 {
		t.Fatalf("expected the region translated by rc's origin, got %+v ok=%v", r, ok)
	}
	if _, ok := h.at(1, 1); ok {
		t.Fatalf("expected the un-translated (pre-origin) coordinates to miss")
	}
}

func TestRenderCtxHovered(t *testing.T) {
	rc := renderCtx{hoverKind: hitTab, hoverIdx: 2}
	if !rc.hovered(hitTab, 2) {
		t.Errorf("expected hovered(hitTab, 2) to be true")
	}
	if rc.hovered(hitTab, 1) || rc.hovered(hitDiffFile, 2) {
		t.Errorf("expected hovered to require both kind and idx to match")
	}
}

// TestCenterOffsetsMatchesLipglossPlace guards against centerOffsets ever
// drifting from lipgloss.Place(Center, Center, ...)'s own arithmetic — the
// whole reason it exists is to compute the exact same offset placeOverlay
// needs, without a second, hand-derived copy of that math.
func TestCenterOffsetsMatchesLipglossPlace(t *testing.T) {
	cases := []struct {
		width, height, boxW, boxH int
	}{
		{80, 24, 56, 12},
		{81, 25, 50, 11}, // odd gaps exercise the rounding
		{100, 40, 52, 9},
		{56, 12, 56, 12}, // exact fit, zero gap
		{40, 10, 56, 12}, // box larger than canvas
	}
	for _, c := range cases {
		box := strings.Repeat(strings.Repeat("x", c.boxW)+"\n", c.boxH)
		box = strings.TrimSuffix(box, "\n")
		placed := lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, box)

		wantY, wantX := -1, -1
		for y, line := range strings.Split(placed, "\n") {
			if x := strings.Index(line, "x"); x >= 0 {
				wantY, wantX = y, x
				break
			}
		}

		gotX, gotY := centerOffsets(c.width, c.height, c.boxW, c.boxH)
		if gotX != wantX || gotY != wantY {
			t.Errorf("centerOffsets(%d,%d,%d,%d) = (%d,%d), want (%d,%d) per lipgloss.Place",
				c.width, c.height, c.boxW, c.boxH, gotX, gotY, wantX, wantY)
		}
	}
}
