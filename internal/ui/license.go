package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

func renderLicenseList(m *Model) string {
	if m.licenseShowAll {
		return renderLicenseFullList(m)
	}
	return renderLicenseSummary(m)
}

// ── Summary view (default) ────────────────────────────────────────────────────

func renderLicenseSummary(m *Model) string {
	var sb strings.Builder

	total := len(m.licensePackages)
	flagged := countNonPermissive(m.licensePackages)

	sb.WriteString(styleTitle.Render("  License Audit") + "\n")
	if flagged > 0 {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("  %d packages scanned", total)) +
			"  " + styleWarning.Render(fmt.Sprintf("⚠  %d need attention", flagged)) + "\n")
	} else {
		sb.WriteString(styleEligibilityOK.Render(fmt.Sprintf("  %d packages scanned — all clear", total)) + "\n")
	}
	sb.WriteString("\n")

	// Category summary table
	var counts [4]int
	for _, p := range m.licensePackages {
		counts[p.Category]++
	}
	sb.WriteString(renderCategoryTable(counts))
	sb.WriteString("\n")

	// Flagged packages list
	visible := m.licenseFiltered // already filtered to non-permissive
	if len(visible) == 0 {
		sb.WriteString(styleEligibilityOK.Render("  No packages need attention.") + "\n")
	} else {
		sb.WriteString(styleMuted.Render("  Packages needing attention:") + "\n\n")
		start, end := m.licenseListWindow(len(visible))
		for i := start; i < end; i++ {
			sb.WriteString(renderFlaggedRow(visible[i], i == m.licenseCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(visible) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(visible)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderLicenseSummaryBar(m))
	return sb.String()
}

func renderCategoryTable(counts [4]int) string {
	type row struct {
		cat  pkg.LicenseCategory
		icon string
		desc string
	}
	rows := []row{
		{
			pkg.LicenseCategoryPermissive,
			"✓",
			"Use freely. Include license files when distributing your software.",
		},
		{
			pkg.LicenseCategoryWeakCopyleft,
			"⚠",
			"OK as a dependency. If you modify the library itself, those changes must be open-sourced.",
		},
		{
			pkg.LicenseCategoryStrongCopyleft,
			"✗",
			"Distributing or serving your app requires open-sourcing it under the same license.",
		},
		{
			pkg.LicenseCategoryUnknown,
			"?",
			"No license found. All rights reserved by default — contact the author before using.",
		},
	}

	var sb strings.Builder
	for _, r := range rows {
		count := counts[r.cat]
		if count == 0 {
			continue
		}
		badge, style := licenseBadge(r.cat)
		icon := style.Render(badge)
		label := style.Render(fmt.Sprintf("%-18s", r.cat.Label()))
		countStr := styleMuted.Render(fmt.Sprintf("%4d  ", count))
		desc := styleMuted.Render(r.desc)
		sb.WriteString(fmt.Sprintf("  %s %s %s%s\n", icon, label, countStr, desc))
	}
	return sb.String()
}

// renderFlaggedRow renders a single row in the needs-attention list, with a
// plain-English description of what action is required.
func renderFlaggedRow(p *pkg.LicenseInfo, isCursor bool) string {
	badge, style := licenseBadge(p.Category)
	icon := style.Render(badge)
	name := truncate(p.Name, 28)
	version := padRight(p.Version, 8)
	lic := style.Render(padRight(licenseDisplay(p.License), 14))
	action := styleMuted.Render(licenseActionText(p))

	row := fmt.Sprintf("  %s %-28s %-8s %-14s %s", icon, name, version, lic, action)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

func licenseActionText(p *pkg.LicenseInfo) string {
	switch p.Category {
	case pkg.LicenseCategoryStrongCopyleft:
		switch {
		case strings.HasPrefix(p.License, "AGPL"):
			return "Distributing or running as a service requires open-sourcing your app"
		case strings.HasPrefix(p.License, "SSPL"):
			return "Using in any service offering requires open-sourcing your infrastructure"
		default:
			return "Distributing your app requires open-sourcing it under the same license"
		}
	case pkg.LicenseCategoryWeakCopyleft:
		switch {
		case strings.HasPrefix(p.License, "LGPL"):
			return "Modifications to this library must be open-sourced; your own code is unaffected"
		case strings.HasPrefix(p.License, "MPL"):
			return "Modified files of this library must be open-sourced; your own code is unaffected"
		default:
			return "Modifications to this library must be open-sourced"
		}
	default: // Unknown
		if p.License == "" {
			return "No license found — contact the author before using"
		}
		return "Unrecognised license — review manually"
	}
}

// ── Full list view ────────────────────────────────────────────────────────────

func renderLicenseFullList(m *Model) string {
	var sb strings.Builder

	total := len(m.licensePackages)
	flagged := countNonPermissive(m.licensePackages)

	sb.WriteString(styleTitle.Render("  License Audit — all packages") + "\n")
	if flagged > 0 {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("  %d packages", total)) +
			"  " + styleWarning.Render(fmt.Sprintf("⚠  %d need attention", flagged)) + "\n")
	} else {
		sb.WriteString(styleMuted.Render(fmt.Sprintf("  %d packages  ✓ all clear", total)) + "\n")
	}
	sb.WriteString("\n")

	sb.WriteString(renderLicenseColumnHeaders())
	sb.WriteString("\n")

	visible := m.licenseFiltered
	if len(visible) == 0 {
		sb.WriteString(styleMuted.Render("  Nothing to show.") + "\n")
	} else {
		start, end := m.licenseListWindow(len(visible))
		for i := start; i < end; i++ {
			sb.WriteString(renderLicenseRow(visible[i], i == m.licenseCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(visible) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(visible)-end)) + "\n")
		}
	}

	if m.licenseFilterActive {
		sb.WriteString("\n" + styleMuted.Render("  /"+m.licenseFilterQuery+"_") + "\n")
	}
	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderLicenseFullBar(m))
	return sb.String()
}

func renderLicenseColumnHeaders() string {
	cols := fmt.Sprintf("  %-3s %-32s %-10s %-18s %-16s %s",
		"", "Package", "Version", "License", "Category", "Dir")
	return styleMuted.Render(cols)
}

func renderLicenseRow(p *pkg.LicenseInfo, isCursor bool) string {
	badge, badgeStyle := licenseBadge(p.Category)
	icon := badgeStyle.Render(badge)
	name := truncate(p.Name, 32)
	version := padRight(p.Version, 10)
	licenseStr := badgeStyle.Render(padRight(licenseDisplay(p.License), 18))
	category := badgeStyle.Render(padRight(p.Category.Label(), 16))
	dir := styleDirTag.Render("(" + p.DirName + ") " + string(p.DepType))

	row := fmt.Sprintf("  %s %-32s %-10s %-18s %-16s %s", icon, name, version, licenseStr, category, dir)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

// ── Status bars ───────────────────────────────────────────────────────────────

func renderLicenseSummaryBar(m *Model) string {
	bar := "j/k: navigate  a: show all packages  q: quit"
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

func renderLicenseFullBar(m *Model) string {
	hints := []string{"j/k: navigate", "/: filter", "s: sort", "a: summary view", "q: quit"}
	bar := strings.Join(hints, "  ")
	if m.licenseFilterActive {
		bar = "type to filter  esc: cancel"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(bar)
}

// ── Window / helpers ──────────────────────────────────────────────────────────

func (m *Model) licenseListWindow(total int) (start, end int) {
	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.licenseScroll
	if m.licenseCursor < start {
		start = m.licenseCursor
	}
	if m.licenseCursor >= start+maxVisible {
		start = m.licenseCursor - maxVisible + 1
	}
	m.licenseScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}

func countNonPermissive(packages []*pkg.LicenseInfo) int {
	n := 0
	for _, p := range packages {
		if p.Category != pkg.LicenseCategoryPermissive {
			n++
		}
	}
	return n
}

func licenseDisplay(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func licenseBadge(cat pkg.LicenseCategory) (string, lipgloss.Style) {
	switch cat {
	case pkg.LicenseCategoryPermissive:
		return "✓", styleEligibilityOK
	case pkg.LicenseCategoryWeakCopyleft:
		return "⚠", styleWarning
	case pkg.LicenseCategoryStrongCopyleft:
		return "✗", styleMajor
	default:
		return "?", styleMuted
	}
}
