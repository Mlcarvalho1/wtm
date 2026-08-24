package gitops

import "strings"

// Status is the derived dirty/ahead/behind state of a worktree relative to its upstream.
type Status struct {
	Dirty       bool
	Ahead       int
	Behind      int
	HasUpstream bool
}

// GetStatus inspects the worktree at path: uncommitted changes (dirty) and
// ahead/behind counts relative to its upstream, if any.
func GetStatus(path string) (Status, error) {
	var st Status

	out, err := runGit(path, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return st, err
	}
	st.Dirty = strings.TrimSpace(out) != ""

	upstream, err := runGit(path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		// No upstream configured — leave Ahead/Behind at 0.
		return st, nil
	}
	st.HasUpstream = strings.TrimSpace(upstream) != ""
	if !st.HasUpstream {
		return st, nil
	}

	counts, err := runGit(path, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err != nil {
		return st, err
	}
	behind, ahead := 0, 0
	fields := strings.Fields(counts)
	if len(fields) == 2 {
		behind = atoiOrZero(fields[0])
		ahead = atoiOrZero(fields[1])
	}
	st.Behind = behind
	st.Ahead = ahead
	return st, nil
}

func atoiOrZero(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
