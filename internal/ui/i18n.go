package ui

import (
	"fmt"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// i18nItem is a row in the combined i18n audit list.
type i18nItem struct {
	key       *pkg.I18nKey
	undefined bool // true = called in source but missing from locale; false = in locale but never called
}

func renderI18nList(m *Model) string {
	var sb strings.Builder

	result := m.i18nResult
	if result == nil {
		return ""
	}

	sb.WriteString(styleTitle.Render("  i18n Audit") + "\n")

	if len(result.LocaleFiles) == 0 {
		sb.WriteString(styleWarning.Render("  No locale files found. Expects JSON files inside a directory named locales/, i18n/, translations/, or lang/.") + "\n")
		sb.WriteString(renderI18nStatusBar(m))
		return sb.String()
	}

	sb.WriteString(styleMuted.Render(fmt.Sprintf(
		"  %d key%s defined  ·  %d locale file%s  ·  %d template%s scanned",
		result.TotalDefined, pluralS(result.TotalDefined),
		len(result.LocaleFiles), pluralS(len(result.LocaleFiles)),
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

	if m.i18nFilterActive {
		sb.WriteString(styleHeader.Render(fmt.Sprintf("  Filter: %s_", m.i18nFilterQuery)) + "\n")
	}

	if len(m.i18nItems) == 0 {
		if m.i18nFilterQuery != "" {
			sb.WriteString(styleMuted.Render("  No keys match the filter.") + "\n")
		} else {
			sb.WriteString(styleEligibilityOK.Render("  All locale keys are used and all source references are defined.") + "\n")
		}
	} else {
		sb.WriteString(renderI18nColumnHeaders() + "\n")
		start, end := m.i18nListWindow(len(m.i18nItems))
		for i := start; i < end; i++ {
			sb.WriteString(renderI18nItem(m.i18nItems[i], i == m.i18nCursor) + "\n")
		}
		if start > 0 {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		if end < len(m.i18nItems) {
			sb.WriteString(styleScrollIndicator.Render(fmt.Sprintf("  ↓ %d more below", len(m.i18nItems)-end)) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleWarning.Render("  "+m.statusMsg) + "\n")
	}

	sb.WriteString(renderI18nStatusBar(m))
	return sb.String()
}

func renderI18nColumnHeaders() string {
	return styleMuted.Render(fmt.Sprintf("  %-2s %-40s %-44s %s", "", "Key", "File", "Line"))
}

func renderI18nItem(item i18nItem, isCursor bool) string {
	var indicator, key string
	if item.undefined {
		indicator = styleMajor.Render("?")
		key = styleMajor.Render(truncate(item.key.Key, 40))
	} else {
		indicator = styleWarning.Render("~")
		key = styleWarning.Render(truncate(item.key.Key, 40))
	}

	file := truncate(item.key.File, 44)
	line := ""
	if item.key.Line > 0 {
		line = fmt.Sprintf("%d", item.key.Line)
	}

	row := fmt.Sprintf("  %s %-40s %-44s %s", indicator, key, file, line)
	if isCursor {
		return styleCursor.Render(row)
	}
	return row
}

func renderI18nStatusBar(m *Model) string {
	legend := "~:unused  ?:undefined"
	hints := legend + "   j/k: navigate  /: filter  q: quit"
	if m.width-2 < len(hints) {
		hints = "j/k:nav  /:filter  q:quit"
	}
	return "\n" + styleStatusBar.Width(m.width).Render(hints)
}

func (m *Model) i18nListWindow(total int) (start, end int) {
	maxVisible := m.height - 14
	if maxVisible < 5 {
		maxVisible = 5
	}
	if total <= maxVisible {
		return 0, total
	}
	start = m.i18nScroll
	if m.i18nCursor < start {
		start = m.i18nCursor
	}
	if m.i18nCursor >= start+maxVisible {
		start = m.i18nCursor - maxVisible + 1
	}
	m.i18nScroll = start
	end = start + maxVisible
	if end > total {
		end = total
	}
	return start, end
}
