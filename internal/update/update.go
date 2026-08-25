// Package update checks GitHub for newer wtm releases and reinstalls via
// `go install` when one is found.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Mlcarvalho1/wtm/internal/version"
)

// apiBase is GitHub's API host; overridden in tests to point at a local
// httptest.Server.
var apiBase = "https://api.github.com"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// CheckLatest queries GitHub for wtm's newest published release, falling
// back to the newest semver-looking tag if no release has been published.
// It returns "" (with a nil error) if the repo has neither yet.
func CheckLatest() (string, error) {
	v, ok, err := latestRelease()
	if err != nil {
		return "", err
	}
	if ok {
		return v, nil
	}
	return latestTag()
}

func latestRelease() (string, bool, error) {
	var body struct {
		TagName string `json:"tag_name"`
	}
	status, err := getJSON(apiBase+"/repos/"+version.RepoSlug+"/releases/latest", &body)
	if err != nil {
		return "", false, err
	}
	if status == http.StatusNotFound {
		return "", false, nil
	}
	if status != http.StatusOK {
		return "", false, fmt.Errorf("github releases API returned status %d", status)
	}
	tag := strings.TrimPrefix(body.TagName, "v")
	if !version.IsValid(tag) {
		return "", false, nil
	}
	return tag, true, nil
}

func latestTag() (string, error) {
	var tags []struct {
		Name string `json:"name"`
	}
	status, err := getJSON(apiBase+"/repos/"+version.RepoSlug+"/tags", &tags)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("github tags API returned status %d", status)
	}

	latest := ""
	for _, t := range tags {
		v := strings.TrimPrefix(t.Name, "v")
		if !version.IsValid(v) {
			continue
		}
		if latest == "" || version.Compare(v, latest) > 0 {
			latest = v
		}
	}
	return latest, nil
}

func getJSON(url string, out any) (int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "wtm-update/"+version.Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// Install reinstalls wtm at the given version (without a leading "v") via
// `go install`, streaming its output to stdout/stderr.
func Install(v string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found on PATH — install Go and re-run 'wtm update' (see install.sh)")
	}

	target := version.ModulePath + "/cmd/wtm@v" + v
	cmd := exec.Command("go", "install", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s: %w", target, err)
	}
	return nil
}
