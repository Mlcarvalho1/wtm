package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Mlcarvalho1/wtm/internal/config"
	"github.com/Mlcarvalho1/wtm/internal/gitops"
	"github.com/Mlcarvalho1/wtm/internal/session"
	"github.com/Mlcarvalho1/wtm/internal/watch"
)

// Worktree is one row: git worktree info + status, joined with its tmux
// session's live state (classified by internal/watch).
type Worktree struct {
	RepoName string
	RepoPath string
	Branch   string
	Path     string

	Dirty  bool
	Ahead  int
	Behind int

	Session  string
	State    watch.State
	LastPane string
}

var sessionNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// SessionName derives a stable tmux session name for a repo+branch pair.
func SessionName(repoName, branch string) string {
	safe := sessionNameSanitizer.ReplaceAllString(repoName+"-"+branch, "-")
	return "wtm-" + strings.Trim(safe, "-")
}

// LoadWorktrees aggregates git worktree + status + tmux session state across
// every repo registered in cfg. Best-effort: a repo that fails to list is
// skipped with its error appended, rather than aborting the whole fleet view.
func LoadWorktrees(cfg config.Config) ([]Worktree, []error) {
	var rows []Worktree
	var errs []error

	for _, repo := range cfg.Repos {
		infos, err := gitops.ListWorktrees(repo.Path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", repo.Name, err))
			continue
		}
		for _, info := range infos {
			branch := info.Branch
			if branch == "" {
				branch = "(detached)"
			}
			row := Worktree{
				RepoName: repo.Name,
				RepoPath: repo.Path,
				Branch:   branch,
				Path:     info.Path,
			}

			if st, err := gitops.GetStatus(info.Path); err == nil {
				row.Dirty = st.Dirty
				row.Ahead = st.Ahead
				row.Behind = st.Behind
			}

			name := SessionName(repo.Name, branch)
			row.Session = name
			if session.Exists(name) {
				row.State = watch.StateRunning
				if pane, err := session.CapturePane(name); err == nil {
					row.LastPane = pane
				}
			} else {
				row.State = watch.StateStopped
			}

			rows = append(rows, row)
		}
	}

	return rows, errs
}
