package ui

import (
	"fmt"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// envItem is a row in the combined env audit list.
type envItem struct {
	v         *pkg.EnvVar
	undefined bool // true = referenced in source but missing from .env.example; false = defined but never referenced
}

func renderEnvList(m *Model) string {
	var sb strings.Builder

	result := m.envResult
	if result == nil {
		return ""
	}

	sb.WriteString(styleTitle.Render("  Env Audit") + "\n")

	if len(result.EnvFiles) == 0 {
		sb.WriteString(styleWarning.Render("  No .env.example files found. Expects .env.example, .env.sample, .env.template, .env.defaults, or .env.schema.") + "\n")
		sb.WriteString(renderEnvStatusBar(m))
		return sb.String()
	}

	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d var%s defined  ·  %d env file%s  ·  %d source file%s scanned",
		result.TotalDefined, pluralS(result.TotalDefined),
		len(result.EnvFiles), pluralS(len(result.EnvFiles)),
		result.SrcFileCount, pluralS(result.SrcFileCount),
	)) + "\n")

	nUnused := len(result.Unused)
	nUndef := len(result.Undefined)
	if nUnused > 0 || nUndef > 0 {
		var findings []string
		if nUnused > 0 {
			findings = append(findings, styleWarning.Render(fmt.Sprintf("%d defined but unused", nUnused)))
		}
		if nUndef > 0 {
			findings = append(findings, styleMajor.Render(fmt.Sprintf("%d used but undefined", nUndef)))
		}
		sb.WriteString(styleMuted.Render("  ") + strings.Join(findings, styleMuted.Render("  ·  ")) + "\n")
	}

	sb.WriteString("\n")

	if m.envFilterActive {
		sb.WriteString(styleHeader.Render(fmt.Sprintf("  Filter: %s_", m.envFilterQuery)) + "\n")
	}

	if len(m.envItems) == 0 {
		if m.envFilterQuery != "" {
			sb.WriteString(styleMuted.Render("  No variables match the filter.") + "\n")
		} else {
			sb.WriteString(styleEligibilityOK.Render("  All env vars are used and all source references are defined.") + "\n")
		}
	} else {
		sb.WriteString(renderEnvColumnHeaders() + "\n")
		start, end := m.envListWindow(len(m.envItems))
		for i := start; i < end; i++ {
			sb.WriteString(renderEnvItem(m.envItems[i], i == m.envCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(m.envItems) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.envItems)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderEnvStatusBar(m))
	return sb.String()
}

func renderEnvColumnHeaders() string {
	return styleMuted.Render(fmt.Sprintf("  %-2s %-40s %-44s %s", "", "Variable", "File", "Line"))
}

func renderEnvItem(item envItem, isCursor bool) string {
	var indicator, key string
	if item.undefined {
		indicator = styleMajor.Render("?")
		key = styleMajor.Render(truncate(item.v.Key, 40))
	} else {
		indicator = styleWarning.Render("~")
		key = styleWarning.Render(truncate(item.v.Key, 40))
	}

	file := truncate(item.v.File, 44)
	line := ""
	if item.v.Line > 0 {
		line = fmt.Sprintf("%d", item.v.Line)
	}

	row := fmt.Sprintf("  %s %-40s %-44s %s", indicator, key, file, line)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

func renderEnvStatusBar(m *Model) string {
	legend := "~:unused  ?:undefined"
	hints := legend + "   j/k: navigate  /: filter  q: quit"
	if m.width-2 < len(hints) {
		hints = "j/k:nav  /:filter  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(hints)
}

func (m *Model) envListWindow(total int) (start, end int) {
	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.envScroll
	if m.envCursor < start {
		start = m.envCursor
	}
	if m.envCursor >= start+maxVisible {
		start = m.envCursor - maxVisible + 1
	}
	m.envScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
