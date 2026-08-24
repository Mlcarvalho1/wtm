package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGetStatus_CleanNoUpstream(t *testing.T) {
	repo := scratchRepo(t)

	st, err := GetStatus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if st.Dirty {
		t.Errorf("expected clean, got dirty")
	}
	if st.HasUpstream {
		t.Errorf("expected no upstream")
	}
	if st.Ahead != 0 || st.Behind != 0 {
		t.Errorf("expected 0/0 ahead/behind, got ahead=%d behind=%d", st.Ahead, st.Behind)
	}
}

func TestGetStatus_Dirty(t *testing.T) {
	repo := scratchRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := GetStatus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Dirty {
		t.Errorf("expected dirty")
	}
}

func TestGetStatus_AheadBehind(t *testing.T) {
	// bare "remote" repo
	remote := t.TempDir()
	runIn(t, remote, "init", "-q", "--bare", "-b", "main")

	repo := scratchRepo(t)
	runIn(t, repo, "remote", "add", "origin", remote)
	runIn(t, repo, "push", "-q", "-u", "origin", "main")

	// Commit locally without pushing, so repo is ahead of its upstream.
	if err := os.WriteFile(filepath.Join(repo, "file2.txt"), []byte("ahead\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runIn(t, repo, "add", "file2.txt")
	runIn(t, repo, "commit", "-q", "-m", "ahead commit")

	st, err := GetStatus(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasUpstream {
		t.Fatalf("expected upstream to be set")
	}
	if st.Ahead != 1 {
		t.Errorf("expected ahead=1, got %d", st.Ahead)
	}
	if st.Behind != 0 {
		t.Errorf("expected behind=0, got %d", st.Behind)
	}
}
