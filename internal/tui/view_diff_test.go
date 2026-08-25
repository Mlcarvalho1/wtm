package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/Mlcarvalho1/wtm/internal/activity"
	"github.com/Mlcarvalho1/wtm/internal/gitops"
)

// TestMain forces the default renderer's color profile to TrueColor for the
// whole package's test run. lipgloss auto-detects the profile from stdout,
// which under `go test` isn't a TTY — left alone, every styled Render() call
// silently degrades to plain, uncolored text, which would make any
// assertion on ANSI/background codes (see
// TestRenderDiffLineBackgroundsAreDistinct) pass or fail based on how the
// tests happen to be invoked rather than on the styling logic itself.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

// TestRenderDiffLineBackgroundsAreDistinct guards against the per-line-kind
// backgrounds silently regressing to "no background" or converging on the
// same color, which would make the diff pane look unchanged from before —
// exactly the kind of thing that's easy to miss just eyeballing a render,
// since these are deliberately subtle, dark tints.
func TestRenderDiffLineBackgroundsAreDistinct(t *testing.T) {
	kinds := []gitops.DiffLineKind{gitops.DiffAdd, gitops.DiffDel, gitops.DiffHunkHeader, gitops.DiffContext}
	seen := map[string]gitops.DiffLineKind{}
	for _, k := range kinds {
		out := renderDiffLine(gitops.DiffLine{Kind: k, LineNo: 1, Text: "x"}, 40)
		bg, ok := findBackgroundColor(out)
		if k == gitops.DiffContext {
			if ok {
				t.Errorf("context line: expected no background, got %q", bg)
			}
			continue
		}
		if !ok {
			t.Fatalf("kind %v: expected a background color in %q", k, out)
		}
		if other, dup := seen[bg]; dup {
			t.Errorf("kind %v shares background %s with kind %v — they'll look identical", k, bg, other)
		}
		seen[bg] = k
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct non-context backgrounds, got %d: %v", len(seen), seen)
	}
}

// findBackgroundColor extracts the first `48;2;r;g;b` truecolor background
// SGR sequence from s, if any. Note the color triple itself contains
// semicolons, so the terminator to search for is the SGR-ending 'm', not
// the first ';'.
func findBackgroundColor(s string) (string, bool) {
	idx := strings.Index(s, "48;2;")
	if idx < 0 {
		return "", false
	}
	end := strings.IndexByte(s[idx:], 'm')
	if end < 0 {
		return "", false
	}
	return s[idx : idx+end], true
}

func TestRenderAttachedTabDetachButtonRegion(t *testing.T) {
	wt := Worktree{Branch: "feat/x", Session: "wtm-repo-feat-x"}
	h := &hitMap{}
	rc := renderCtx{hits: h, x0: 3, y0: 5}
	out := renderAttachedTab(wt, "frame line 1\nframe line 2", rc)

	// The hint (and its button region) is rc's row 1, one below the head
	// line, at the same x-origin.
	r, ok := h.at(3, 6)
	if !ok || r.kind != hitDetachButton {
		t.Fatalf("expected a hitDetachButton region at the hint line, got %+v ok=%v", r, ok)
	}
	lines := strings.Split(out, "\n")
	plain := ansi.Strip(lines[1])
	if !strings.Contains(plain, "ctrl-b d to detach") {
		t.Errorf("expected the detach hint text on the button's own line, got %q", plain)
	}
}

func TestRenderSidebarRegisterHitRegions(t *testing.T) {
	rows := []Worktree{
		{RepoName: "a", Branch: "main"},
		{RepoName: "a", Branch: "feat/x"},
	}
	h := &hitMap{}
	rc := renderCtx{hits: h, x0: 0, y0: 3, hoverKind: hitNone, hoverIdx: -1}
	renderSidebar(rows, 1, 30, 20, rc)

	// Row 0 ("main") occupies the group header line, then its own two
	// lines; row 1 ("feat/x", the cursor) follows immediately after.
	if r, ok := h.at(5, 4); !ok || r.kind != hitSidebarRow || r.idx != 0 {
		t.Fatalf("expected row 0's first line to hit-test to idx 0, got %+v ok=%v", r, ok)
	}
	if r, ok := h.at(5, 6); !ok || r.kind != hitSidebarRow || r.idx != 1 {
		t.Fatalf("expected row 1's first line to hit-test to idx 1, got %+v ok=%v", r, ok)
	}
}

func TestRenderDiffTabScrollClamps(t *testing.T) {
	wt := Worktree{Base: "main"}
	files := []gitops.FileDiff{{Path: "a.go", Adds: 1, Dels: 1}}
	lines := make([]gitops.DiffLine, 50)
	for i := range lines {
		lines[i] = gitops.DiffLine{Kind: gitops.DiffContext, LineNo: i + 1, Text: "line"}
	}
	rc := renderCtx{hits: &hitMap{}}

	for _, scroll := range []int{-100, 0, 5, 1000} {
		out := renderDiffTab(wt, files, 0, lines, false, nil, 80, 20, scroll, rc)
		if out == "" {
			t.Fatalf("scroll=%d: expected non-empty render", scroll)
		}
	}
}

func TestRenderDiffTabFileClickRegion(t *testing.T) {
	wt := Worktree{Base: "main"}
	files := []gitops.FileDiff{
		{Path: "a.go", Adds: 1, Dels: 1},
		{Path: "b.go", Adds: 2, Dels: 0},
	}
	h := &hitMap{}
	rc := renderCtx{hits: h, x0: 2, y0: 5}
	renderDiffTab(wt, files, 0, nil, true, nil, 80, 20, 0, rc)

	// File rows start after the "N files +A -D" summary line and a blank
	// line — row 0 is at y=5+2, row 1 immediately below it.
	if r, ok := h.at(3, 7); !ok || r.kind != hitDiffFile || r.idx != 0 {
		t.Fatalf("expected file 0's region, got %+v ok=%v", r, ok)
	}
	if r, ok := h.at(3, 8); !ok || r.kind != hitDiffFile || r.idx != 1 {
		t.Fatalf("expected file 1's region, got %+v ok=%v", r, ok)
	}
}

func TestRenderActivityTabScrollClamps(t *testing.T) {
	var entries []activity.Entry
	for range 30 {
		entries = append(entries, activity.Entry{Time: time.Now(), Branch: "feat/x", Text: "did a thing"})
	}
	for _, scroll := range []int{-100, 0, 5, 1000} {
		if out := renderActivityTab(entries, 80, 10, scroll); out == "" {
			t.Fatalf("scroll=%d: expected non-empty render", scroll)
		}
	}
	// Zero entries and zero height are the degenerate cases most likely to
	// panic on an off-by-one slice bound.
	if out := renderActivityTab(nil, 80, 10, 0); out == "" {
		t.Fatalf("expected a placeholder render for no entries")
	}
	renderActivityTab(entries, 80, 0, 0)
}

func TestModalButtonRegionsAlignWithButtonText(t *testing.T) {
	t.Run("confirm", func(t *testing.T) {
		modal, buttons := renderConfirmModal("Remove worktree foo?", []string{"step one", "step two"}, hitNone)
		requireButtonTextAt(t, modal, buttons, hitModalPrimary, "y — confirm")
		requireButtonTextAt(t, modal, buttons, hitModalSecondary, "n — cancel")
	})

	t.Run("new worktree", func(t *testing.T) {
		modal, buttons := renderNewWorktreeModal("repo", false, "feat/x", "origin/main", false,
			"path preview", "session preview", hitNone)
		requireButtonTextAt(t, modal, buttons, hitModalPrimary, "enter — create")
		requireButtonTextAt(t, modal, buttons, hitModalSecondary, "esc — cancel")
	})

	t.Run("palette", func(t *testing.T) {
		modal, buttons := renderPaletteModal("", [][2]string{{"Refresh git status", "r"}, {"Remove worktree", "x"}}, 0, -1)
		if len(buttons) != 2 {
			t.Fatalf("expected one region per command row, got %d", len(buttons))
		}
		requireTextAtRegion(t, modal, buttons[0], "Refresh git status")
		requireTextAtRegion(t, modal, buttons[1], "Remove worktree")
	})
}

// requireButtonTextAt finds the region of the given kind and asserts the
// modal's plain text at its origin starts with want.
func requireButtonTextAt(t *testing.T, modal string, buttons []hitRegion, kind hitKind, want string) {
	t.Helper()
	for _, b := range buttons {
		if b.kind == kind {
			requireTextAtRegion(t, modal, b, want)
			return
		}
	}
	t.Fatalf("no region of kind %v returned", kind)
}

// requireTextAtRegion asserts modal's plain (ANSI-stripped) text starting at
// region's (x0, y0) begins with want — i.e. the hit region actually sits on
// top of the text it's supposed to make clickable, not just somewhere in
// the modal.
func requireTextAtRegion(t *testing.T, modal string, region hitRegion, want string) {
	t.Helper()
	lines := strings.Split(modal, "\n")
	if region.y0 < 0 || region.y0 >= len(lines) {
		t.Fatalf("region y0=%d out of range (modal has %d lines)", region.y0, len(lines))
	}
	plain := ansi.Strip(lines[region.y0])
	runes := []rune(plain)
	if region.x0 < 0 || region.x0+len([]rune(want)) > len(runes) {
		t.Fatalf("region x0=%d, want %q doesn't fit in line %q", region.x0, want, plain)
	}
	got := string(runes[region.x0 : region.x0+len([]rune(want))])
	if got != want {
		t.Errorf("at (%d,%d): got %q, want %q\nfull line: %q", region.x0, region.y0, got, want, plain)
	}
}
