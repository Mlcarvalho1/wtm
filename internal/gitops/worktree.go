// Package gitops shells out to the real git CLI for worktree management.
// git is the source of truth; we never reimplement its object model.
package gitops

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// WorktreeInfo is one entry from `git worktree list --porcelain`.
type WorktreeInfo struct {
	Path        string
	Head        string
	Branch      string // short form, e.g. "main" (empty if detached)
	Detached    bool
	Bare        bool
	Locked      bool
	LockReason  string
	Prunable    bool
	PruneReason string
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// ListWorktrees returns all worktrees registered against the repo at repoPath
// (any worktree path, or the main checkout, works as repoPath).
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	out, err := runGit(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

func parseWorktreeList(out string) []WorktreeInfo {
	var result []WorktreeInfo
	var cur *WorktreeInfo

	flush := func() {
		if cur != nil {
			result = append(result, *cur)
			cur = nil
		}
	}

	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			flush()
			continue
		}
		field, rest, _ := strings.Cut(line, " ")
		switch field {
		case "worktree":
			flush()
			cur = &WorktreeInfo{Path: rest}
		case "HEAD":
			if cur != nil {
				cur.Head = rest
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(rest, "refs/heads/")
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
				cur.LockReason = rest
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
				cur.PruneReason = rest
			}
		}
	}
	flush()
	return result
}

// AddWorktree creates a new worktree at path.
// If createBranch is true, branch is created from baseRef (git worktree add -b branch path baseRef).
// If createBranch is false, branch must already exist and is checked out as-is.
func AddWorktree(repoPath, path, branch, baseRef string, createBranch bool) error {
	args := []string{"worktree", "add"}
	if createBranch {
		args = append(args, "-b", branch, path)
		if baseRef != "" {
			args = append(args, baseRef)
		}
	} else {
		args = append(args, path, branch)
	}
	_, err := runGit(repoPath, args...)
	return err
}

// RemoveWorktree removes a worktree. force removes it even with uncommitted changes.
func RemoveWorktree(repoPath, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := runGit(repoPath, args...)
	return err
}

// DeleteBranch deletes a local branch. force uses -D instead of -d.
func DeleteBranch(repoPath, branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := runGit(repoPath, "branch", flag, branch)
	return err
}
