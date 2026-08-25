package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// plainAt returns the ANSI-stripped rune slice at row y of s, or fails the
// test if y is out of range.
func plainAt(t *testing.T, s string, y int) []rune {
	t.Helper()
	lines := strings.Split(s, "\n")
	if y < 0 || y >= len(lines) {
		t.Fatalf("row %d out of range (%d lines)", y, len(lines))
	}
	return []rune(ansi.Strip(lines[y]))
}

func TestRenderHeaderCommandsButtonRegion(t *testing.T) {
	h := &hitMap{}
	rc := renderCtx{hits: h}
	out := renderHeader(80, fleetCounts{running: 1, needs: 2, idle: 3, stopped: 4}, rc)

	plain := plainAt(t, out, 0)
	found := false
	for x := 0; x < 80; x++ {
		r, ok := h.at(x, 0)
		if !ok || r.kind != hitCommandsButton {
			continue
		}
		found = true
		if got := string(plain[r.x0:r.x1]); got != ": commands" {
			t.Errorf("region text = %q, want %q", got, ": commands")
		}
		break
	}
	if !found {
		t.Fatalf("expected a hitCommandsButton region somewhere in the header line %q", string(plain))
	}
}

func TestRenderFleetBarChipRegions(t *testing.T) {
	h := &hitMap{}
	rc := renderCtx{hits: h}
	rows := []needsInputRow{{Branch: "fix/webhook-retry", Idx: 1}, {Branch: "main", Idx: 4}}
	out := renderFleetBar(80, rows, rc)
	plain := plainAt(t, out, 0)

	for _, row := range rows {
		found := false
		for x := 0; x < 80; x++ {
			if rg, ok := h.at(x, 0); ok && rg.kind == hitSidebarRow && rg.idx == row.Idx {
				found = true
				got := string(plain[rg.x0:rg.x1])
				if !strings.Contains(got, row.Branch) {
					t.Errorf("chip region for idx=%d text = %q, want it to contain %q", row.Idx, got, row.Branch)
				}
				break
			}
		}
		if !found {
			t.Errorf("no chip region registered for idx=%d (%s)", row.Idx, row.Branch)
		}
	}
}

func TestRenderTabStripRegions(t *testing.T) {
	h := &hitMap{}
	rc := renderCtx{hits: h}
	out := renderTabStrip(tabDiff, "~/wt/repo/branch", 100, rc)
	plain := plainAt(t, out, 0)

	labels := map[tabKind]string{tabSession: "session", tabDiff: "diff", tabActivity: "activity"}
	for kind, label := range labels {
		found := false
		for x := 0; x < 100; x++ {
			if rg, ok := h.at(x, 0); ok && rg.kind == hitTab && rg.idx == int(kind) {
				found = true
				got := string(plain[rg.x0:rg.x1])
				if got != label {
					t.Errorf("tab %v region text = %q, want %q", kind, got, label)
				}
				break
			}
		}
		if !found {
			t.Errorf("no region registered for tab %v", kind)
		}
	}
}
