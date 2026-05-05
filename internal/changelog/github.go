package changelog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/marcelmettler/bumpit/internal/auth"
)

const githubAPIBase = "https://api.github.com"

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// FetchGitHubChangelog fetches releases between fromVersion and toVersion for owner/repo.
// Returns the combined markdown, the number of matching releases, and whether breaking
// changes were detected. Authentication is resolved automatically via auth.GitHubToken.
func FetchGitHubChangelog(owner, repo, fromVersion, toVersion string) (markdown string, releaseCount int, hasBreaking bool, err error) {
	client := &http.Client{Timeout: 15 * time.Second}

	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=50", githubAPIBase, owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", 0, false, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "bumpit-cli/1.0")
	if token := auth.GitHubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, false, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return "", 0, false, fmt.Errorf("repository %s/%s not found", owner, repo)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return "", 0, false, fmt.Errorf("GitHub rate limit reached — set GITHUB_TOKEN for higher limits (or log in via gh CLI)")
	default:
		return "", 0, false, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 20<<20)).Decode(&releases); err != nil {
		return "", 0, false, fmt.Errorf("decode releases: %w", err)
	}

	filtered := filterReleases(releases, fromVersion, toVersion)
	if len(filtered) == 0 {
		return "No release notes found between versions.", 0, false, nil
	}

	var parts []string
	for _, r := range filtered {
		header := fmt.Sprintf("## %s", r.TagName)
		if r.Name != "" && r.Name != r.TagName {
			header += " — " + r.Name
		}
		date := r.PublishedAt.Format("2006-01-02")
		header += fmt.Sprintf(" (%s)", date)
		body := strings.TrimSpace(r.Body)
		if body == "" {
			body = "_No release notes._"
		}
		if containsBreaking(body) || containsBreaking(r.Name) {
			hasBreaking = true
		}
		parts = append(parts, header+"\n\n"+body)
	}

	markdown = strings.Join(parts, "\n\n---\n\n")
	return markdown, len(filtered), hasBreaking, nil
}

func filterReleases(releases []githubRelease, from, to string) []githubRelease {
	fromSV, err1 := semver.NewVersion(from)
	toSV, err2 := semver.NewVersion(to)

	var result []githubRelease
	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		tag := strings.TrimPrefix(r.TagName, "v")
		rv, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}

		var include bool
		if err1 == nil && err2 == nil {
			include = rv.GreaterThan(fromSV) && (rv.LessThan(toSV) || rv.Equal(toSV))
		} else {
			include = r.TagName == to || r.TagName == "v"+to
		}

		if include {
			result = append(result, r)
		}
	}
	return result
}

func containsBreaking(s string) bool {
	upper := strings.ToUpper(s)
	return strings.Contains(upper, "BREAKING CHANGE") ||
		strings.Contains(upper, "BREAKING:") ||
		strings.Contains(s, "⚠️") ||
		strings.Contains(upper, "INCOMPATIBLE")
}
