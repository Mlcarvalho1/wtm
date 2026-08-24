package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseNumstat(t *testing.T) {
	out := "38\t12\tsrc/payments/tap.ts\n21\t0\tsrc/payments/reader.ts\n"
	files := parseNumstat(out)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
	if files[0] != (FileDiff{Path: "src/payments/tap.ts", Adds: 38, Dels: 12}) {
		t.Errorf("unexpected first file: %+v", files[0])
	}
	if files[1] != (FileDiff{Path: "src/payments/reader.ts", Adds: 21, Dels: 0}) {
		t.Errorf("unexpected second file: %+v", files[1])
	}
}

func TestParseNumstat_Binary(t *testing.T) {
	files := parseNumstat("-\t-\tassets/logo.png\n")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %+v", len(files), files)
	}
	if files[0].Path != "assets/logo.png" || files[0].Adds != 0 || files[0].Dels != 0 {
		t.Errorf("unexpected binary file entry: %+v", files[0])
	}
}

func TestParseUnifiedDiff(t *testing.T) {
	out := `diff --git a/foo.go b/foo.go
index abc123..def456 100644
--- a/foo.go
+++ b/foo.go
@@ -10,3 +10,4 @@ func foo() {
 	a := 1
-	b := 2
+	b := 3
+	c := 4
 	return a
`
	lines := parseUnifiedDiff(out)

	var got []DiffLine
	for _, l := range lines {
		got = append(got, l)
	}

	want := []DiffLine{
		{Kind: DiffHunkHeader, Text: "@@ -10,3 +10,4 @@ func foo() {"},
		{Kind: DiffContext, LineNo: 10, Text: "\ta := 1"},
		{Kind: DiffDel, LineNo: 11, Text: "\tb := 2"},
		{Kind: DiffAdd, LineNo: 11, Text: "\tb := 3"},
		{Kind: DiffAdd, LineNo: 12, Text: "\tc := 4"},
		{Kind: DiffContext, LineNo: 13, Text: "\treturn a"},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d lines, got %d: %+v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDiffFiles_And_DiffFile(t *testing.T) {
	repo := scratchRepo(t)
	wtPath := filepath.Join(t.TempDir(), "feat-diff")
	if err := AddWorktree(repo, wtPath, "feat/diff", "main", true); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("scratch\nmore\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runIn(t, wtPath, "commit", "-aqm", "extend readme")
	if err := os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runIn(t, wtPath, "add", "new.txt")
	runIn(t, wtPath, "commit", "-qm", "add new file")

	files, err := DiffFiles(wtPath, "main")
	if err != nil {
		t.Fatalf("DiffFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %d: %+v", len(files), files)
	}

	found := false
	for _, f := range files {
		if f.Path == "new.txt" {
			found = true
			if f.Adds != 1 || f.Dels != 0 {
				t.Errorf("new.txt: expected +1/-0, got +%d/-%d", f.Adds, f.Dels)
			}
		}
	}
	if !found {
		t.Fatalf("expected new.txt in diff, got %+v", files)
	}

	dlines, err := DiffFile(wtPath, "main", "new.txt")
	if err != nil {
		t.Fatalf("DiffFile: %v", err)
	}
	var adds int
	for _, l := range dlines {
		if l.Kind == DiffAdd {
			adds++
		}
	}
	if adds != 1 {
		t.Errorf("expected 1 added line in new.txt diff, got %d: %+v", adds, dlines)
	}
}

func TestDefaultBaseRef(t *testing.T) {
	t.Run("falls back to local main", func(t *testing.T) {
		repo := scratchRepo(t)
		ref, err := DefaultBaseRef(repo)
		if err != nil {
			t.Fatal(err)
		}
		if ref != "main" {
			t.Errorf("expected main, got %q", ref)
		}
	})

	t.Run("prefers upstream", func(t *testing.T) {
		remote := t.TempDir()
		runIn(t, remote, "init", "-q", "--bare", "-b", "main")
		repo := scratchRepo(t)
		runIn(t, repo, "remote", "add", "origin", remote)
		runIn(t, repo, "push", "-q", "-u", "origin", "main")

		ref, err := DefaultBaseRef(repo)
		if err != nil {
			t.Fatal(err)
		}
		if ref != "origin/main" {
			t.Errorf("expected origin/main, got %q", ref)
		}
	})
}

func TestMerge(t *testing.T) {
	repo := scratchRepo(t)
	wtPath := filepath.Join(t.TempDir(), "feat-merge")
	if err := AddWorktree(repo, wtPath, "feat/merge", "main", true); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runIn(t, wtPath, "add", "new.txt")
	runIn(t, wtPath, "commit", "-qm", "add new file")

	if err := Merge(repo, "feat/merge"); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "new.txt")); err != nil {
		t.Errorf("expected new.txt to exist in repo after merge: %v", err)
	}
}
