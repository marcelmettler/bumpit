package pnpm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/marcelmettler/chorekit/internal/detect"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

// installedPkgJSON is the subset of a node_modules package.json used for licenses.
type installedPkgJSON struct {
	Version  string          `json:"version"`
	License  json.RawMessage `json:"license"`  // string or {"type":"..."}
	Licenses []struct {
		Type string `json:"type"`
	} `json:"licenses"` // legacy array format
}

// FindLicenses returns license information for every direct dependency in the
// package.json at dir. Data is read from locally installed node_modules; packages
// that are not installed are reported with an empty license string and Unknown category.
func FindLicenses(dir, root string) ([]*pkg.LicenseInfo, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var pkgJSON packageJSONDeps
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	dirName := detect.ShortName(dir, root)
	seen := make(map[string]bool)
	var result []*pkg.LicenseInfo

	add := func(name string, depType pkg.DepType) {
		if seen[name] {
			return
		}
		seen[name] = true
		license, version := readInstalledLicense(name, dir, root)
		result = append(result, &pkg.LicenseInfo{
			Name:     name,
			Version:  version,
			License:  license,
			Category: classifyLicense(license),
			Dir:      dir,
			DirName:  dirName,
			Source:   "npm",
			DepType:  depType,
		})
	}

	for name := range pkgJSON.Dependencies {
		add(name, pkg.DepDependencies)
	}
	for name := range pkgJSON.DevDependencies {
		add(name, pkg.DepDevDependencies)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category // risky (low int) first
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// readInstalledLicense looks up the license and version of name from node_modules.
// It checks the package-local node_modules first, then the workspace root (hoisted).
func readInstalledLicense(name, dir, root string) (license, version string) {
	for _, base := range []string{dir, root} {
		data, err := os.ReadFile(nmPkgJSON(base, name))
		if err != nil {
			continue
		}
		var raw installedPkgJSON
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		return extractLicenseString(raw), raw.Version
	}
	return "", ""
}

// nmPkgJSON returns the path to <base>/node_modules/<name>/package.json,
// correctly handling scoped packages like @scope/pkg.
func nmPkgJSON(base, name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name[1:], "/", 2)
		if len(parts) == 2 {
			return filepath.Join(base, "node_modules", "@"+parts[0], parts[1], "package.json")
		}
	}
	return filepath.Join(base, "node_modules", name, "package.json")
}

// extractLicenseString normalises all license field variants to a plain string.
func extractLicenseString(raw installedPkgJSON) string {
	if len(raw.License) > 0 && raw.License[0] != 'n' {
		var s string
		if err := json.Unmarshal(raw.License, &s); err == nil {
			return strings.TrimSpace(s)
		}
		var obj struct{ Type string }
		if err := json.Unmarshal(raw.License, &obj); err == nil && obj.Type != "" {
			return strings.TrimSpace(obj.Type)
		}
	}
	if len(raw.Licenses) > 0 {
		return strings.TrimSpace(raw.Licenses[0].Type)
	}
	return ""
}

// classifyLicense maps an SPDX expression to a risk category.
// It handles compound expressions:
//
//	"MIT OR GPL-3.0-or-later" → consumer picks the most permissive → Permissive
//	"MIT AND Apache-2.0"      → all must be complied with → most restrictive wins
func classifyLicense(license string) pkg.LicenseCategory {
	if license == "" {
		return pkg.LicenseCategoryUnknown
	}
	return classifySPDX(license)
}

// classifySPDX recursively evaluates an SPDX expression.
// Category int values are ordered so that higher = more permissive:
//
//	0 StrongCopyleft < 1 Unknown < 2 WeakCopyleft < 3 Permissive
func classifySPDX(expr string) pkg.LicenseCategory {
	expr = strings.TrimSpace(expr)
	// Strip a single layer of outer parentheses.
	if len(expr) >= 2 && expr[0] == '(' && expr[len(expr)-1] == ')' {
		expr = strings.TrimSpace(expr[1 : len(expr)-1])
	}

	// OR — consumer chooses: take the most permissive branch.
	if parts := splitSPDXOp(expr, " OR "); len(parts) > 1 {
		best := pkg.LicenseCategoryStrongCopyleft
		for _, p := range parts {
			if c := classifySPDX(p); c > best {
				best = c
			}
		}
		return best
	}

	// AND — all licenses apply: take the most restrictive.
	if parts := splitSPDXOp(expr, " AND "); len(parts) > 1 {
		worst := pkg.LicenseCategoryPermissive
		for _, p := range parts {
			if c := classifySPDX(p); c < worst {
				worst = c
			}
		}
		return worst
	}

	// Leaf identifier. Strip "WITH <exception>" clause (e.g. GPL-2.0 WITH Classpath-exception-2.0).
	id := expr
	if i := strings.Index(id, " WITH "); i != -1 {
		id = id[:i]
	}
	// Normalise legacy "+" suffix and "-only" / "-or-later" variants.
	id = strings.TrimSuffix(id, "+")
	id = strings.TrimSuffix(strings.TrimSuffix(id, "-only"), "-or-later")
	id = strings.TrimSpace(id)

	switch {
	case strongCopyleftLicenses[id]:
		return pkg.LicenseCategoryStrongCopyleft
	case weakCopyleftLicenses[id]:
		return pkg.LicenseCategoryWeakCopyleft
	case permissiveLicenses[id]:
		return pkg.LicenseCategoryPermissive
	default:
		return pkg.LicenseCategoryUnknown
	}
}

// splitSPDXOp splits expr by sep only at the top level (not inside parentheses).
// Returns nil if sep is not present at the top level.
func splitSPDXOp(expr, sep string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth == 0 && strings.HasPrefix(expr[i:], sep) {
				parts = append(parts, strings.TrimSpace(expr[start:i]))
				start = i + len(sep)
				i += len(sep) - 1
			}
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return append(parts, strings.TrimSpace(expr[start:]))
}

var permissiveLicenses = map[string]bool{
	"MIT": true, "MIT-0": true,
	"ISC": true,
	"BSD-2-Clause": true, "BSD-3-Clause": true, "BSD-4-Clause": true,
	"Apache-2.0": true,
	"0BSD": true,
	"Unlicense": true,
	"WTFPL": true,
	"CC0-1.0": true,
	"CC-BY-4.0": true, "CC-BY-3.0": true, // attribution required, no copyleft
	"BlueOak-1.0.0": true,
	"Zlib": true,
	"Python-2.0": true,
	"PSF-2.0": true,
	"Artistic-2.0": true,
	"Beerware": true,
	// Font licenses — permissive for embedding/use in software
	"OFL-1.1": true, "OFL-1.1-RFN": true, "OFL-1.1-no-RFN": true,
}

var weakCopyleftLicenses = map[string]bool{
	"LGPL-2.0": true, "LGPL-2.1": true, "LGPL-3.0": true,
	"MPL-2.0": true,
	"CDDL-1.0": true,
	"EPL-1.0": true, "EPL-2.0": true,
	"EUPL-1.1": true, "EUPL-1.2": true,
	"OSL-3.0": true,
}

var strongCopyleftLicenses = map[string]bool{
	"GPL-2.0": true, "GPL-3.0": true,
	"AGPL-3.0": true,
	"SSPL-1.0": true,
	"BUSL-1.1": true,
	"CC-BY-SA-4.0": true,
}
