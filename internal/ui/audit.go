package ui

import (
	"fmt"
	"strings"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

func renderAuditList(m *Model) string {
	var sb strings.Builder

	result := m.auditResult
	if result == nil {
		return ""
	}

	sb.WriteString(styleTitle.Render("  Security Audit") + "\n")

	total := result.Critical + result.High + result.Moderate + result.Low + result.Info
	if total == 0 {
		sb.WriteString(styleEligibilityOK.Render("  No vulnerabilities found.") + "\n")
		sb.WriteString(renderAuditStatusBar(m))
		return sb.String()
	}

	// Severity summary line.
	var parts []string
	if result.Critical > 0 {
		parts = append(parts, styleCritical.Render(fmt.Sprintf("%d critical", result.Critical)))
	}
	if result.High > 0 {
		parts = append(parts, styleMajor.Render(fmt.Sprintf("%d high", result.High)))
	}
	if result.Moderate > 0 {
		parts = append(parts, styleWarning.Render(fmt.Sprintf("%d moderate", result.Moderate)))
	}
	if result.Low > 0 {
		parts = append(parts, styleMinor.Render(fmt.Sprintf("%d low", result.Low)))
	}
	if result.Info > 0 {
		parts = append(parts, styleMuted.Render(fmt.Sprintf("%d info", result.Info)))
	}
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  %d vulnerabilit%s:  ", total, pluralIes(total)))+
		strings.Join(parts, styleMuted.Render("  "))+"\n")

	sb.WriteString("\n")

	if m.auditFilterActive {
		sb.WriteString(styleHeader.Render(fmt.Sprintf("  Filter: %s_", m.auditFilterQuery)) + "\n")
	}

	items := m.auditFiltered
	if len(items) == 0 {
		if m.auditFilterQuery != "" {
			sb.WriteString(styleMuted.Render("  No vulnerabilities match the filter.") + "\n")
		}
	} else {
		start, end := m.auditListWindow(len(items))
		for i := start; i < end; i++ {
			renderAuditItem(&sb, items[i], i == m.auditCursor)
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(items) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(items)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderAuditStatusBar(m))
	return sb.String()
}

func renderAuditItem(sb *strings.Builder, v *pkg.Vuln, isCursor bool) {
	sev := auditSevLabel(v.Severity)
	name := padRight(truncate(v.PackageName, 22), 22)
	vulnRange := padRight(truncate(v.VulnerableVersions, 13), 13)

	var fixPart string
	if v.Fixable {
		fixPart = styleEligibilityOK.Render("→ " + truncate(v.PatchedVersions, 13))
	} else {
		fixPart = styleMajor.Render("→ no fix      ")
	}

	title := truncate(v.Title, 38)

	line1 := fmt.Sprintf("  %s  %s  %s  %s  %s", sev, name, vulnRange, fixPart, title)

	// Second line: CVEs and dependency path.
	var cveStr string
	if len(v.CVEs) > 0 {
		shown := v.CVEs
		suffix := ""
		if len(shown) > 2 {
			shown = shown[:2]
			suffix = fmt.Sprintf(" +%d", len(v.CVEs)-2)
		}
		cveStr = strings.Join(shown, ", ") + suffix + "  ·  "
	}
	line2 := fmt.Sprintf("        %s%s", cveStr, auditShortPath(v))

	if isCursor {
		sb.WriteString(styleCursor.Render(line1) + "\n")
		sb.WriteString(styleCursor.Render(line2) + "\n\n")
	} else {
		sb.WriteString(line1 + "\n")
		sb.WriteString(styleMuted.Render(line2) + "\n\n")
	}
}

func auditSevLabel(severity string) string {
	switch severity {
	case "critical":
		return styleCritical.Render("CRIT")
	case "high":
		return styleMajor.Render("HIGH")
	case "moderate":
		return styleWarning.Render("MOD ")
	case "low":
		return styleMinor.Render("LOW ")
	default:
		return styleMuted.Render("INFO")
	}
}

func auditShortPath(v *pkg.Vuln) string {
	if v.IsDirect {
		return "(direct)"
	}
	if len(v.Paths) == 0 {
		return ""
	}
	// Prefer the shortest path.
	shortest := v.Paths[0]
	for _, p := range v.Paths[1:] {
		if len(p) < len(shortest) {
			shortest = p
		}
	}
	path := truncate(shortest, 60)
	extra := len(v.Paths) - 1
	if extra > 0 {
		path += fmt.Sprintf("  +%d path%s", extra, pluralS(extra))
	}
	return path
}

func renderAuditStatusBar(m *Model) string {
	hints := "j/k: navigate  /: filter  q: quit"
	if m.width-2 < len(hints) {
		hints = "j/k:nav  /:filter  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(hints)
}

func (m *Model) auditListWindow(total int) (start, end int) {
	// 3 display lines per item: row1, row2, blank line.
	maxVisible := (m.height - 10) / 3
	if maxVisible < 3 {
		maxVisible = 3
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.auditScroll
	if m.auditCursor < start {
		start = m.auditCursor
	}
	if m.auditCursor >= start+maxVisible {
		start = m.auditCursor - maxVisible + 1
	}
	m.auditScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
