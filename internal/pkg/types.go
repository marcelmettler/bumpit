package pkg

import (
	"fmt"
	"time"
)

// UpdateKind classifies a version bump.
type UpdateKind int

const (
	KindPatch   UpdateKind = iota
	KindMinor
	KindMajor
	KindUnknown
)

func (k UpdateKind) String() string {
	switch k {
	case KindPatch:
		return "patch"
	case KindMinor:
		return "minor"
	case KindMajor:
		return "MAJOR"
	default:
		return "unknown"
	}
}

// DepType classifies the dependency relationship.
type DepType string

const (
	DepDependencies     DepType = "dep"
	DepDevDependencies  DepType = "devDep"
	DepPeerDependencies DepType = "peer"
	DepIndirect         DepType = "indirect"
)

// Highlights holds key information extracted from changelog markdown sections.
type Highlights struct {
	Breaking  []string
	Features  []string
	Fixes     []string
	Migration []string
}

func (h Highlights) IsEmpty() bool {
	return len(h.Breaking)+len(h.Features)+len(h.Fixes)+len(h.Migration) == 0
}

// PackageUpdate holds all information about an outdated package.
type PackageUpdate struct {
	// Identity
	Name    string
	Dir     string // absolute path to the package file dir
	DirName string // short display name for directory
	Source  string // "npm" | "go"

	// Versions
	Current string
	Wanted  string
	Latest  string
	Kind    UpdateKind

	// Metadata from registry
	PublishedAt   time.Time
	RepositoryURL string
	DepType       DepType

	// Changelog
	Changelog        string // full markdown
	ChangelogFetched bool
	HasBreaking      bool
	OneLineSummary   string
	Highlights       Highlights

	// Eligibility
	EligibleAt time.Time // zero means always eligible
	IsEligible bool

	// Security
	VulnCount int

	// UI state
	Selected bool

	// Render cache — populated lazily by the detail view
	renderedChangelog string
	renderedWidth     int
}

// CachedRender returns the cached glamour-rendered changelog, or "" if stale.
func (p *PackageUpdate) CachedRender(width int) string {
	if p.renderedWidth == width {
		return p.renderedChangelog
	}
	return ""
}

// SetCachedRender stores the rendered changelog for the given width.
func (p *PackageUpdate) SetCachedRender(rendered string, width int) {
	p.renderedChangelog = rendered
	p.renderedWidth = width
}

// CSSClass represents a CSS class selector defined in a stylesheet.
type CSSClass struct {
	Name string
	File string // relative path to the stylesheet
	Line int
}

// ArtifactDir represents a generated or installed directory that can be safely deleted.
type ArtifactDir struct {
	Path     string // absolute path
	RelPath  string // path relative to workspace root, used for display
	Kind     string // human label: "dependencies", "build output", "cache", etc.
	Size     int64  // total size in bytes
	Selected bool
}

// LicenseCategory classifies the risk level of a license for commercial use.
type LicenseCategory int

const (
	LicenseCategoryStrongCopyleft LicenseCategory = iota // GPL-*, AGPL-*
	LicenseCategoryUnknown                               // missing or unrecognised
	LicenseCategoryWeakCopyleft                          // LGPL-*, MPL-2.0
	LicenseCategoryPermissive                            // MIT, ISC, Apache-2.0, BSD-*
)

func (c LicenseCategory) Label() string {
	switch c {
	case LicenseCategoryPermissive:
		return "permissive"
	case LicenseCategoryWeakCopyleft:
		return "weak copyleft"
	case LicenseCategoryStrongCopyleft:
		return "strong copyleft"
	default:
		return "unknown"
	}
}

// LicenseInfo holds license information for a single installed dependency.
type LicenseInfo struct {
	Name     string
	Version  string
	License  string          // raw SPDX expression from package.json; "" if not installed
	Category LicenseCategory
	Dir      string
	DirName  string
	Source   string // "npm" | "go"
	DepType  DepType
}

// UnusedPackage holds information about a direct dependency not imported anywhere in the project.
type UnusedPackage struct {
	Name     string
	Dir      string
	DirName  string
	Source   string // "npm" | "go"
	DepType  DepType
	Selected bool
}

// I18nKey represents a translation key — either defined in a locale file or referenced in source code.
type I18nKey struct {
	Key  string // dot-notation key, e.g. "common.save"
	File string // relative path
	Line int    // 0 for locale file entries (JSON line numbers not tracked)
}

// TodoItem represents a TODO/FIXME/HACK/XXX comment found in source code.
type TodoItem struct {
	Kind string // "TODO", "FIXME", "HACK", "XXX"
	Text string // comment text after the keyword
	File string // relative path
	Line int
}

// AgeDisplay returns a human-readable age string.
func (p *PackageUpdate) AgeDisplay() string {
	if p.PublishedAt.IsZero() {
		return "unknown"
	}
	d := time.Since(p.PublishedAt)
	switch {
	case d < 24*time.Hour:
		return "< 1 day"
	case d < 48*time.Hour:
		return "1 day"
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
}
