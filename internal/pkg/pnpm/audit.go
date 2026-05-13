package pnpm

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

type auditJSON struct {
	Advisories map[string]advisoryJSON `json:"advisories"`
	Metadata   struct {
		Vulnerabilities struct {
			Info     int `json:"info"`
			Low      int `json:"low"`
			Moderate int `json:"moderate"`
			High     int `json:"high"`
			Critical int `json:"critical"`
		} `json:"vulnerabilities"`
	} `json:"metadata"`
}

type advisoryJSON struct {
	ID                 int      `json:"id"`
	ModuleName         string   `json:"module_name"`
	Severity           string   `json:"severity"`
	Title              string   `json:"title"`
	URL                string   `json:"url"`
	VulnerableVersions string   `json:"vulnerable_versions"`
	PatchedVersions    string   `json:"patched_versions"`
	Recommendation     string   `json:"recommendation"`
	CVEs               []string `json:"cves"`
	Findings           []struct {
		Version string   `json:"version"`
		Paths   []string `json:"paths"`
	} `json:"findings"`
}

// Audit runs `pnpm audit --json` in dir and returns the full parsed result.
func Audit(dir string) (*pkg.AuditResult, error) {
	cmd := exec.Command("pnpm", "audit", "--json")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run() // exits non-zero when vulnerabilities are found — not an error

	var raw auditJSON
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return &pkg.AuditResult{}, nil
	}

	result := &pkg.AuditResult{
		Critical: raw.Metadata.Vulnerabilities.Critical,
		High:     raw.Metadata.Vulnerabilities.High,
		Moderate: raw.Metadata.Vulnerabilities.Moderate,
		Low:      raw.Metadata.Vulnerabilities.Low,
		Info:     raw.Metadata.Vulnerabilities.Info,
	}

	for _, adv := range raw.Advisories {
		vuln := &pkg.Vuln{
			ID:                 adv.ID,
			Title:              adv.Title,
			Severity:           adv.Severity,
			PackageName:        adv.ModuleName,
			VulnerableVersions: adv.VulnerableVersions,
			PatchedVersions:    adv.PatchedVersions,
			Recommendation:     adv.Recommendation,
			CVEs:               adv.CVEs,
			URL:                adv.URL,
			Fixable:            auditIsFixable(adv.PatchedVersions),
		}

		seen := make(map[string]bool)
		for _, finding := range adv.Findings {
			if vuln.InstalledVersion == "" {
				vuln.InstalledVersion = finding.Version
			}
			for _, path := range finding.Paths {
				if !seen[path] {
					seen[path] = true
					vuln.Paths = append(vuln.Paths, path)
					if !strings.Contains(path, ">") {
						vuln.IsDirect = true
					}
				}
			}
		}

		result.Vulns = append(result.Vulns, vuln)
	}

	sort.Slice(result.Vulns, func(i, j int) bool {
		a, b := result.Vulns[i], result.Vulns[j]
		oa, ob := auditSevOrder(a.Severity), auditSevOrder(b.Severity)
		if oa != ob {
			return oa < ob
		}
		if a.Fixable != b.Fixable {
			return a.Fixable // fixable first within same severity
		}
		return a.PackageName < b.PackageName
	})

	return result, nil
}

func auditIsFixable(patchedVersions string) bool {
	s := strings.TrimSpace(patchedVersions)
	return s != "" && s != "<0.0.0" && s != "<0.0.0-0"
}

func auditSevOrder(s string) int {
	switch s {
	case "critical":
		return 0
	case "high":
		return 1
	case "moderate":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
