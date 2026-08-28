// Package version holds wtm's version metadata and semver comparison used
// by `wtm version` and `wtm update`.
package version

import (
	"strconv"
	"strings"
)

// Version is wtm's current release version (semver, no leading "v"). Bump
// this and tag the matching commit (git tag vX.Y.Z && git push origin
// vX.Y.Z) when cutting a new release, so `wtm update` can find it.
const Version = "0.0.1"

// ModulePath is wtm's Go module path, used to reinstall via `go install`.
const ModulePath = "github.com/Mlcarvalho1/wtm"

// RepoSlug is wtm's GitHub "owner/repo", used to check for new releases.
const RepoSlug = "Mlcarvalho1/wtm"

// IsValid reports whether v looks like a dotted major.minor.patch version,
// with an optional "-pre" style suffix on any component.
func IsValid(v string) bool {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		p, _, _ = strings.Cut(p, "-")
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// Compare returns -1, 0, or 1 as semver a is less than, equal to, or
// greater than b. Callers should check IsValid first; an unparseable
// component is treated as 0.
func Compare(a, b string) int {
	pa, pb := parseParts(a), parseParts(b)
	for i := 0; i < 3; i++ {
		switch {
		case pa[i] < pb[i]:
			return -1
		case pa[i] > pb[i]:
			return 1
		}
	}
	return 0
}

func parseParts(v string) [3]int {
	var out [3]int
	parts := strings.SplitN(v, ".", 3)
	for i := 0; i < len(parts) && i < 3; i++ {
		p, _, _ := strings.Cut(parts[i], "-")
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}
