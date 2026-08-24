package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// scratchRepo creates a throwaway git repo with one commit on "main" and
// returns its path. Every test gets its own temp dir via t.TempDir().
func scratchRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "initial commit")

	return dir
}

func TestListWorktrees_MainOnly(t *testing.T) {
	repo := scratchRepo(t)

	wts, err := ListWorktrees(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d: %+v", len(wts), wts)
	}
	if wts[0].Branch != "main" {
		t.Errorf("expected branch main, got %q", wts[0].Branch)
	}
	if wts[0].Detached {
		t.Errorf("expected not detached")
	}
}

func TestAddListRemoveWorktree(t *testing.T) {
	repo := scratchRepo(t)
	wtPath := filepath.Join(t.TempDir(), "feat-worktree")

	if err := AddWorktree(repo, wtPath, "feat/x", "main", true); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	wts, err := ListWorktrees(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d: %+v", len(wts), wts)
	}

	var found *WorktreeInfo
	for i := range wts {
		if wts[i].Branch == "feat/x" {
			found = &wts[i]
		}
	}
	if found == nil {
		t.Fatalf("did not find feat/x worktree in %+v", wts)
	}
	// git worktree list may resolve symlinks (e.g. /tmp -> /private/tmp on macOS).
	resolvedWant, _ := filepath.EvalSymlinks(wtPath)
	resolvedGot, _ := filepath.EvalSymlinks(found.Path)
	if resolvedGot != resolvedWant {
		t.Errorf("expected path %q, got %q", resolvedWant, resolvedGot)
	}

	if err := RemoveWorktree(repo, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	wts, err = ListWorktrees(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree after removal, got %d: %+v", len(wts), wts)
	}

	if err := DeleteBranch(repo, "feat/x", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
}

func TestAddWorktree_ExistingBranch(t *testing.T) {
	repo := scratchRepo(t)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("branch", "existing-branch")

	wtPath := filepath.Join(t.TempDir(), "existing-wt")
	if err := AddWorktree(repo, wtPath, "existing-branch", "", false); err != nil {
		t.Fatalf("AddWorktree with existing branch: %v", err)
	}

	wts, err := ListWorktrees(repo)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, wt := range wts {
		if wt.Branch == "existing-branch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected to find existing-branch worktree, got %+v", wts)
	}
}

func TestRemoveWorktree_DirtyRequiresForce(t *testing.T) {
	repo := scratchRepo(t)
	wtPath := filepath.Join(t.TempDir(), "dirty-wt")
	if err := AddWorktree(repo, wtPath, "feat/dirty", "main", true); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtPath, "README.md"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveWorktree(repo, wtPath, false); err == nil {
		t.Fatalf("expected RemoveWorktree without force to fail on dirty worktree")
	}
	if err := RemoveWorktree(repo, wtPath, true); err != nil {
		t.Fatalf("RemoveWorktree with force: %v", err)
	}
}
