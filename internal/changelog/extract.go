package changelog

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/maece/bumpit/internal/pkg"
)

type sectionKind int

const (
	sectionUnknown  sectionKind = iota
	sectionBreaking             // "Breaking Changes", "⚠️ Breaking", "Incompatible"
	sectionFeature              // "Features", "Added", "New", "Enhancements"
	sectionFix                  // "Bug Fixes", "Fixed", "Patch"
	sectionMigration            // "Migration Guide", "Upgrade", "How to upgrade"
)

// ExtractHighlights parses markdown changelog text into structured highlights.
// It works on the joined multi-release markdown produced by FetchGitHubChangelog.
func ExtractHighlights(md string) pkg.Highlights {
	var h pkg.Highlights
	var current sectionKind

	for _, line := range strings.Split(md, "\n") {
		stripped := strings.TrimSpace(line)

		// Section headers: ## or ###
		if strings.HasPrefix(stripped, "#") {
			raw := strings.TrimLeft(stripped, "#")
			raw = strings.TrimSpace(raw)
			// Strip emoji and non-ASCII from header text before classifying
			current = classifySection(normaliseHeader(raw))
			continue
		}

		item := extractBulletItem(stripped)
		if item == "" {
			continue
		}

		switch current {
		case sectionBreaking:
			h.Breaking = append(h.Breaking, item)
		case sectionFeature:
			if len(h.Features) < 10 {
				h.Features = append(h.Features, item)
			}
		case sectionFix:
			if len(h.Fixes) < 10 {
				h.Fixes = append(h.Fixes, item)
			}
		case sectionMigration:
			h.Migration = append(h.Migration, item)
		}
	}

	return h
}

// BuildSummary produces the one-line string shown in the list view.
//
// Approach D — composite:
//   - If breaking changes exist: show first item verbatim + overflow count, then feat/fix counts
//   - Otherwise: feat/fix counts, falling back to release count
//
// Examples:
//
//	"⚠ removed ReactDOM.render (+1) · 5 feat · 3 fix"
//	"5 feat · 3 fix"
//	"3 releases"
func BuildSummary(h pkg.Highlights, releaseCount int) string {
	var parts []string

	if len(h.Breaking) > 0 {
		first := truncItem(h.Breaking[0], 48)
		s := "⚠ " + first
		if len(h.Breaking) > 1 {
			s += fmt.Sprintf(" (+%d)", len(h.Breaking)-1)
		}
		parts = append(parts, s)
	}
	if n := len(h.Features); n > 0 {
		parts = append(parts, fmt.Sprintf("%d feat", n))
	}
	if n := len(h.Fixes); n > 0 {
		parts = append(parts, fmt.Sprintf("%d fix", n))
	}

	if len(parts) == 0 {
		if releaseCount == 1 {
			return "1 release"
		}
		return fmt.Sprintf("%d releases", releaseCount)
	}
	return strings.Join(parts, " · ")
}

// classifySection maps a normalised header string to a section kind.
func classifySection(h string) sectionKind {
	switch {
	case containsAny(h, "breaking", "incompatible", "removed"):
		return sectionBreaking
	case containsAny(h, "feat", "add", "new", "enhancement", "what"):
		return sectionFeature
	case containsAny(h, "fix", "bug", "patch", "correct"):
		return sectionFix
	case containsAny(h, "migrat", "upgrade", "how to"):
		return sectionMigration
	default:
		return sectionUnknown
	}
}

// normaliseHeader strips emoji, leading punctuation, and lowercases a header.
func normaliseHeader(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r <= 127 && (unicode.IsLetter(r) || unicode.IsSpace(r)) {
			b.WriteRune(unicode.ToLower(r))
		} else if r <= 127 {
			b.WriteRune(' ')
		}
		// skip non-ASCII (emoji, special chars)
	}
	return strings.TrimSpace(b.String())
}

// extractBulletItem strips a leading bullet marker and returns the cleaned text.
// Returns "" if the line is not a bullet item.
func extractBulletItem(line string) string {
	for _, prefix := range []string{"- ", "* ", "+ ", "• "} {
		if strings.HasPrefix(line, prefix) {
			return cleanItem(line[len(prefix):])
		}
	}
	// Numbered list: "1. ", "12. "
	if len(line) >= 3 && line[0] >= '1' && line[0] <= '9' {
		dot := strings.Index(line, ". ")
		if dot > 0 && dot < 4 {
			return cleanItem(line[dot+2:])
		}
	}
	return ""
}

// cleanItem strips inline markdown from a bullet item string.
func cleanItem(s string) string {
	s = strings.TrimSpace(s)
	s = stripMarkdownLinks(s)
	s = stripInlineCode(s)
	// Remove GitHub PR/commit references at end: "(#1234)" or "by @user in #1234"
	if i := strings.Index(s, " by @"); i > 10 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// stripMarkdownLinks replaces [text](url) with text.
func stripMarkdownLinks(s string) string {
	for {
		start := strings.Index(s, "[")
		if start < 0 {
			break
		}
		mid := strings.Index(s[start:], "](")
		if mid < 0 {
			break
		}
		end := strings.Index(s[start+mid+2:], ")")
		if end < 0 {
			break
		}
		text := s[start+1 : start+mid]
		s = s[:start] + text + s[start+mid+2+end+1:]
	}
	return s
}

// stripInlineCode replaces `code` with code.
func stripInlineCode(s string) string {
	return strings.NewReplacer("`", "").Replace(s)
}

func truncItem(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
