package gitops

import (
	"strconv"
	"strings"
)

// FileDiff is one changed file between a worktree's HEAD and its base ref,
// as reported by `git diff --numstat`.
type FileDiff struct {
	Path string
	Adds int
	Dels int
}

// DiffFiles lists the files changed on the worktree at path relative to
// base (merge-base diff: base...HEAD, the same range GitHub uses for a PR
// diff — commits unique to the worktree's branch, not uncommitted changes,
// which are already surfaced separately via Status.Dirty).
func DiffFiles(path, base string) ([]FileDiff, error) {
	out, err := runGit(path, "diff", "--numstat", base+"...HEAD")
	if err != nil {
		return nil, err
	}
	return parseNumstat(out), nil
}

func parseNumstat(out string) []FileDiff {
	var files []FileDiff
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		adds, _ := strconv.Atoi(fields[0]) // "-" for binary files parses to 0
		dels, _ := strconv.Atoi(fields[1])
		files = append(files, FileDiff{Path: fields[2], Adds: adds, Dels: dels})
	}
	return files
}

// DiffLineKind classifies one rendered line of a unified diff hunk.
type DiffLineKind int

const (
	DiffHunkHeader DiffLineKind = iota
	DiffContext
	DiffAdd
	DiffDel
)

// DiffLine is one line of a file's unified diff, pre-classified for
// rendering (color, line-number column) without the caller re-parsing '@@'
// headers or +/- prefixes itself.
type DiffLine struct {
	Kind DiffLineKind
	// LineNo is the relevant side's line number for this line: the new-file
	// number for context/added lines, the old-file number for removed
	// lines. Zero for hunk headers.
	LineNo int
	Text   string
}

// DiffFile returns the unified diff of one file on the worktree at path,
// relative to base, as pre-classified lines ready for a two-pane hunk view.
func DiffFile(path, base, file string) ([]DiffLine, error) {
	out, err := runGit(path, "diff", "--unified=3", base+"...HEAD", "--", file)
	if err != nil {
		return nil, err
	}
	return parseUnifiedDiff(out), nil
}

// hunkHeaderRe-free parse: unified diff hunk headers look like
// "@@ -oldStart,oldCount +newStart,newCount @@ optional context".
func parseUnifiedDiff(out string) []DiffLine {
	var lines []DiffLine
	oldNo, newNo := 0, 0

	for _, raw := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(raw, "@@ "):
			oldNo, newNo = parseHunkHeader(raw)
			lines = append(lines, DiffLine{Kind: DiffHunkHeader, Text: raw})
		case strings.HasPrefix(raw, "diff --git"), strings.HasPrefix(raw, "index "),
			strings.HasPrefix(raw, "--- "), strings.HasPrefix(raw, "+++ "),
			strings.HasPrefix(raw, "new file"), strings.HasPrefix(raw, "deleted file"),
			strings.HasPrefix(raw, "Binary files"):
			// File-level preamble, not a content line — skip.
			continue
		case strings.HasPrefix(raw, "+"):
			lines = append(lines, DiffLine{Kind: DiffAdd, LineNo: newNo, Text: raw[1:]})
			newNo++
		case strings.HasPrefix(raw, "-"):
			lines = append(lines, DiffLine{Kind: DiffDel, LineNo: oldNo, Text: raw[1:]})
			oldNo++
		case strings.HasPrefix(raw, " "):
			lines = append(lines, DiffLine{Kind: DiffContext, LineNo: newNo, Text: raw[1:]})
			oldNo++
			newNo++
		case raw == "":
			// Trailing blank line from the split; drop it rather than
			// render a phantom context line.
		default:
			lines = append(lines, DiffLine{Kind: DiffContext, Text: raw})
		}
	}
	return lines
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	// "@@ -12,7 +12,9 @@ func foo(" -> old="-12,7" new="+12,9"
	fields := strings.Fields(header)
	for _, f := range fields {
		switch {
		case strings.HasPrefix(f, "-"):
			oldStart = firstInt(f[1:])
		case strings.HasPrefix(f, "+"):
			newStart = firstInt(f[1:])
		}
	}
	return oldStart, newStart
}

func firstInt(s string) int {
	comma := strings.IndexByte(s, ',')
	if comma >= 0 {
		s = s[:comma]
	}
	n, _ := strconv.Atoi(s)
	return n
}

// DefaultBaseRef picks the ref a worktree's branch should be reviewed and
// merged against, since git itself doesn't remember the ref a branch was
// created from: the branch's own upstream if it has one, else the repo's
// remote HEAD (origin/main-or-whatever), else a local "main" or "master" if
// either exists.
func DefaultBaseRef(repoPath string) (string, error) {
	if out, err := runGit(repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err == nil {
		if ref := strings.TrimSpace(out); ref != "" {
			return ref, nil
		}
	}
	if out, err := runGit(repoPath, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if ref := strings.TrimSpace(out); ref != "" {
			return ref, nil
		}
	}
	for _, candidate := range []string{"main", "master"} {
		if _, err := runGit(repoPath, "rev-parse", "--verify", "--quiet", candidate); err == nil {
			return candidate, nil
		}
	}
	return "HEAD", nil
}

// Merge merges branch into whatever is currently checked out at repoPath
// (--no-ff, so the merge always leaves a commit even if it could fast-forward
// — this is a review action on the main checkout, not a rebase). The
// worktree the branch lives in, and its session, are untouched.
func Merge(repoPath, branch string) error {
	_, err := runGit(repoPath, "merge", "--no-ff", branch)
	return err
}
