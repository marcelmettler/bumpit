package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const npmRegistryBase = "https://registry.npmjs.org"

// PackageInfo holds relevant metadata from the npm registry.
type PackageInfo struct {
	Name          string
	LatestVersion string
	PublishedAt   time.Time
	RepositoryURL string
}

// npmResponse is the minimal subset of the npm registry response we care about.
type npmResponse struct {
	Name  string                     `json:"name"`
	Time  map[string]string          `json:"time"`
	Versions map[string]npmVersion   `json:"versions"`
	DistTags map[string]string       `json:"dist-tags"`
}

type npmVersion struct {
	Repository *npmRepository `json:"repository"`
}

type npmRepository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// FetchPackageInfo fetches metadata for a package from the npm registry.
func FetchPackageInfo(name, version string) (*PackageInfo, error) {
	url := fmt.Sprintf("%s/%s", npmRegistryBase, name)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch npm registry %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned %d for %s", resp.StatusCode, name)
	}

	var data npmResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode npm response for %s: %w", name, err)
	}

	info := &PackageInfo{
		Name:          data.Name,
		LatestVersion: data.DistTags["latest"],
	}

	// Get publish date for the specific version
	if ts, ok := data.Time[version]; ok {
		t, err := time.Parse(time.RFC3339, ts)
		if err == nil {
			info.PublishedAt = t
		}
	}

	// Get repository URL from the version info
	if vdata, ok := data.Versions[version]; ok && vdata.Repository != nil {
		info.RepositoryURL = cleanRepoURL(vdata.Repository.URL)
	}

	// Fallback: try latest version for repo URL
	if info.RepositoryURL == "" {
		if latest, ok := data.DistTags["latest"]; ok {
			if vdata, ok := data.Versions[latest]; ok && vdata.Repository != nil {
				info.RepositoryURL = cleanRepoURL(vdata.Repository.URL)
			}
		}
	}

	return info, nil
}

// cleanRepoURL normalizes repository URLs to plain HTTPS form.
// Handles: git+https://github.com/owner/repo.git, git://github.com/..., etc.
func cleanRepoURL(raw string) string {
	raw = strings.TrimPrefix(raw, "git+")
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.TrimPrefix(raw, "git://")
	raw = strings.TrimPrefix(raw, "ssh://git@")
	if strings.HasPrefix(raw, "github.com/") {
		raw = "https://" + raw
	}
	return raw
}

// ExtractGitHubOwnerRepo parses a GitHub repository URL and returns owner, repo.
var githubURLRe = regexp.MustCompile(`github\.com[/:]([^/]+)/([^/\s]+?)(?:\.git)?$`)

func ExtractGitHubOwnerRepo(repoURL string) (owner, repo string, ok bool) {
	m := githubURLRe.FindStringSubmatch(repoURL)
	if len(m) < 3 {
		return "", "", false
	}
	return m[1], m[2], true
}

// ParseMinimumReleaseAge parses strings like "3 days", "72h", "3d" into a duration.
func ParseMinimumReleaseAge(s string) time.Duration {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 3 * 24 * time.Hour
	}

	// Try Go duration format first (e.g. "72h")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	// Try "N days" or "N day"
	var n int
	var unit string
	if _, err := fmt.Sscanf(s, "%d %s", &n, &unit); err == nil {
		switch {
		case strings.HasPrefix(unit, "day"):
			return time.Duration(n) * 24 * time.Hour
		case strings.HasPrefix(unit, "hour"):
			return time.Duration(n) * time.Hour
		case strings.HasPrefix(unit, "week"):
			return time.Duration(n) * 7 * 24 * time.Hour
		}
	}

	// Try "Nd" shorthand
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	return 3 * 24 * time.Hour // default
}
