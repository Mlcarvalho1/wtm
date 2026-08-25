package tui

import "math"

// hitKind identifies what a clickable/hoverable screen region represents.
type hitKind int

const (
	hitNone           hitKind = iota
	hitSidebarRow             // idx: index into the visible worktree list
	hitTab                    // idx: tabKind
	hitCommandsButton         // header's ": commands" button
	hitDiffFile               // idx: index into the diff tab's file list
	hitMergeButton
	hitDiscardButton
	hitSplitToggle // diff tab's split/unified view toggle
	hitAttachButton
	hitDetachButton
	hitModalPrimary
	hitModalSecondary
	hitPaletteRow // idx: index into the filtered command list
	hitOverlayBackdrop
)

// hitRegion is one clickable/hoverable rectangle, half-open on both axes:
// x0<=x<x1, y0<=y<y1.
type hitRegion struct {
	x0, y0, x1, y1 int
	kind           hitKind
	idx            int
}

func (r hitRegion) contains(x, y int) bool {
	return x >= r.x0 && x < r.x1 && y >= r.y0 && y < r.y1
}

// hitMap collects every clickable/hoverable region drawn in one frame.
// View() rebuilds it from scratch on every call; a mouse event arriving in
// Update() consults whatever the *previous* frame registered — at most one
// frame (a few milliseconds) stale, which is imperceptible, and far simpler
// than threading a live collector through every render function's return
// path.
type hitMap struct {
	regions []hitRegion
}

func (h *hitMap) reset() {
	h.regions = h.regions[:0]
}

func (h *hitMap) add(kind hitKind, idx, x0, y0, x1, y1 int) {
	if h == nil || x1 <= x0 || y1 <= y0 {
		return
	}
	h.regions = append(h.regions, hitRegion{x0: x0, y0: y0, x1: x1, y1: y1, kind: kind, idx: idx})
}

// at returns the most recently added region containing (x, y), so a region
// registered on top of another (e.g. a modal's own buttons over its
// full-screen backdrop) takes priority over the one beneath it.
func (h *hitMap) at(x, y int) (hitRegion, bool) {
	if h == nil {
		return hitRegion{}, false
	}
	for i := len(h.regions) - 1; i >= 0; i-- {
		if h.regions[i].contains(x, y) {
			return h.regions[i], true
		}
	}
	return hitRegion{}, false
}

// renderCtx carries a render function's mouse plumbing: where its own local
// (0,0) lands in absolute screen cells, the shared collector to register
// regions into (in local coordinates — addHit offsets them), and which
// region (if any) the mouse currently sits over, for hover styling.
type renderCtx struct {
	hits      *hitMap
	x0, y0    int
	hoverKind hitKind
	hoverIdx  int
}

func (rc renderCtx) hovered(kind hitKind, idx int) bool {
	return rc.hoverKind == kind && rc.hoverIdx == idx
}

func (rc renderCtx) addHit(kind hitKind, idx, x0, y0, x1, y1 int) {
	rc.hits.add(kind, idx, rc.x0+x0, rc.y0+y0, rc.x0+x1, rc.y0+y1)
}

// translate returns a copy of rc anchored at a new absolute origin — for
// handing a sub-region (a pane within a pane) its own local coordinate
// space without disturbing the shared hits collector or hover state.
func (rc renderCtx) translate(dx, dy int) renderCtx {
	rc.x0 += dx
	rc.y0 += dy
	return rc
}

// centerOffsets replicates lipgloss.Place(Center, Center, ...)'s centering
// arithmetic so overlay button regions (computed in the modal's own local
// coordinates) can be translated to the same absolute screen position
// centerOverlay renders the modal at, without hand-duplicating a second,
// possibly-drifting copy of the modal's layout.
func centerOffsets(width, height, boxW, boxH int) (x0, y0 int) {
	gapW := max(width-boxW, 0)
	splitW := int(math.Round(float64(gapW) * 0.5))
	x0 = gapW - splitW

	gapH := max(height-boxH, 0)
	splitH := int(math.Round(float64(gapH) * 0.5))
	y0 = gapH - splitH
	return x0, y0
}
