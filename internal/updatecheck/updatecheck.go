// Package updatecheck answers "how does a self-hosted operator find out
// IdpForge 1.1.0 is out while they're running 1.0.0?" -- the running
// instance itself polls GitHub Releases and surfaces it, instead of
// leaving operators to notice on their own.
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// LatestRelease fetches the latest GitHub release for owner/repo via the
// public REST API. No auth token: this only ever reads public release
// metadata, well within GitHub's unauthenticated rate limit for an
// infrequent check.
func LatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "idpforge-update-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: unexpected status %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// IsNewer reports whether latest (a tag like "v1.1.0") is a newer release
// than current (the running build's version string: an actual tag when
// built by the release pipeline, or a "git describe" output like
// "v1.0.2-3-gabcdef" / "dev" for a local build). Anything current can't
// parse as a clean vMAJOR.MINOR.PATCH never claims an update is available,
// since there is nothing meaningful to compare it against.
func IsNewer(current, latest string) bool {
	cur, ok := parseSemver(current)
	if !ok {
		return false
	}
	lat, ok := parseSemver(latest)
	if !ok {
		return false
	}
	for i := range cur {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" || strings.ContainsAny(v, "-+ ") {
		return out, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
