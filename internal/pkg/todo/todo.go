package todo

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// ScanResult holds all TODO/FIXME/HACK/XXX items found in the project.
type ScanResult struct {
	Items []*pkg.TodoItem
}

var skipDirs = map[string]bool{
	"node_modules":     true,
	".git":             true,
	"dist":             true,
	"build":            true,
	"out":              true,
	".next":            true,
	".nuxt":            true,
	".output":          true,
	".angular":         true,
	".svelte-kit":      true,
	".vite":            true,
	".turbo":           true,
	".cache":           true,
	"coverage":         true,
	".nyc_output":      true,
	"storybook-static": true,
}

var scanExts = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".mjs": true, ".cjs": true, ".vue": true, ".svelte": true,
	".css": true, ".scss": true, ".sass": true, ".less": true,
	".html": true, ".yaml": true, ".yml": true, ".toml": true,
	".sh": true, ".bash": true, ".py": true, ".rb": true,
	".rs": true, ".java": true, ".kt": true, ".swift": true,
}

// reKeyword matches TODO/FIXME/HACK/XXX followed by optional colon, paren, or space.
var reKeyword = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX)\b[:(]?(.*)`)

// Scan walks root and returns all TODO-style comments found in source files.
func Scan(root string) (*ScanResult, error) {
	var items []*pkg.TodoItem

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !scanExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		found := scanFile(path, rel)
		items = append(items, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}
		return items[i].Line < items[j].Line
	})

	return &ScanResult{Items: items}, nil
}

func scanFile(absPath, relPath string) []*pkg.TodoItem {
	f, err := os.Open(absPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var items []*pkg.TodoItem
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		m := reKeyword.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[2])
		// Strip trailing comment closers from inline block comments.
		text = strings.TrimRight(text, " \t")
		text = strings.TrimSuffix(text, "-->")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		// Strip a leading colon or closing paren that belongs to TODO(author): patterns.
		text = strings.TrimPrefix(text, ")")
		text = strings.TrimPrefix(text, ":")
		text = strings.TrimSpace(text)
		items = append(items, &pkg.TodoItem{
			Kind: m[1],
			Text: text,
			File: relPath,
			Line: lineNum,
		})
	}
	return items
}
