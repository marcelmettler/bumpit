package pnpm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/marcelmettler/bumpit/internal/detect"
	"github.com/marcelmettler/bumpit/internal/pkg"
)

// validPackageName matches npm package names: optional @scope/name or plain name.
var validPackageName = regexp.MustCompile(`^(@[a-z0-9_\-\.]+/)?[a-z0-9_\-\.]+$`)

// outdatedEntry is the JSON structure returned by `pnpm outdated --json`.
type outdatedEntry struct {
	Current        string `json:"current"`
	Wanted         string `json:"wanted"`
	Latest         string `json:"latest"`
	DependencyType string `json:"dependencyType"`
}

// IsInstalled reports whether pnpm is available in PATH.
func IsInstalled() bool {
	_, err := exec.LookPath("pnpm")
	return err == nil
}

// Outdated runs `pnpm outdated --json` in dir and returns the parsed packages.
func Outdated(dir, root string) ([]*pkg.PackageUpdate, error) {
	cmd := exec.Command("pnpm", "outdated", "--json")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	// pnpm exits with code 1 when there are outdated packages — treat as success
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("pnpm outdated failed: %w\n%s", err, stderr.String())
			}
		} else {
			return nil, fmt.Errorf("pnpm outdated failed: %w", err)
		}
	}

	out := stdout.Bytes()
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, nil // no outdated packages
	}

	var entries map[string]outdatedEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parse pnpm output: %w", err)
	}

	dirName := detect.ShortName(dir, root)
	var updates []*pkg.PackageUpdate
	for name, entry := range entries {
		kind := classifyUpdate(entry.Current, entry.Latest)
		updates = append(updates, &pkg.PackageUpdate{
			Name:       name,
			Dir:        dir,
			DirName:    dirName,
			Source:     "npm",
			Current:    entry.Current,
			Wanted:     entry.Wanted,
			Latest:     entry.Latest,
			Kind:       kind,
			DepType:    mapDepType(entry.DependencyType),
			IsEligible: true, // will be evaluated after registry enrichment
		})
	}

	return updates, nil
}

// RunUpdate executes `pnpm update` for the given packages in the given directory.
func RunUpdate(dir string, packages []string) (string, error) {
	if len(packages) == 0 {
		return "", nil
	}
	for _, name := range packages {
		if !validPackageName.MatchString(name) {
			return "", fmt.Errorf("refusing to update: invalid package name %q", name)
		}
	}
	args := append([]string{"update", "--latest"}, packages...)
	cmd := exec.Command("pnpm", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("pnpm update failed: %w", err)
	}
	return out.String(), nil
}

// AuditPackages runs `pnpm audit --json` and returns a map of package name → vuln count.
func AuditPackages(dir string) (map[string]int, error) {
	cmd := exec.Command("pnpm", "audit", "--json")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// pnpm audit exits non-zero when vulnerabilities are found
	_ = cmd.Run()

	var result struct {
		Advisories map[string]struct {
			ModuleName string `json:"module_name"`
			Severity   string `json:"severity"`
		} `json:"advisories"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, nil // audit may not be available, skip
	}

	counts := make(map[string]int)
	for _, adv := range result.Advisories {
		counts[adv.ModuleName]++
	}
	return counts, nil
}

// NpmrcMinimumReleaseAge reads the minimum-release-age from .npmrc in dir.
// Returns the default "3 days" if not found.
func NpmrcMinimumReleaseAge(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".npmrc"))
	if err != nil {
		return "3 days"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "minimum-release-age") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "3 days"
}

func mapDepType(dt string) pkg.DepType {
	switch dt {
	case "dependencies":
		return pkg.DepDependencies
	case "devDependencies":
		return pkg.DepDevDependencies
	case "peerDependencies":
		return pkg.DepPeerDependencies
	default:
		return pkg.DepDependencies
	}
}

func classifyUpdate(current, latest string) pkg.UpdateKind {
	cur, err1 := semver.NewVersion(current)
	lat, err2 := semver.NewVersion(latest)
	if err1 != nil || err2 != nil {
		return pkg.KindUnknown
	}
	switch {
	case lat.Major() > cur.Major():
		return pkg.KindMajor
	case lat.Minor() > cur.Minor():
		return pkg.KindMinor
	case lat.Patch() > cur.Patch():
		return pkg.KindPatch
	default:
		return pkg.KindPatch
	}
}
