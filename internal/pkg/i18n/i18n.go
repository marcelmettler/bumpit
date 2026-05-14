package i18n

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

// ScanResult holds the outcome of an i18n key audit.
type ScanResult struct {
	Unused       []*pkg.I18nKey // defined in locale files, never called in source
	Undefined    []*pkg.I18nKey // called in source, absent from all locale files
	LocaleFiles  []string       // locale files that were scanned
	TotalDefined int
	SrcFileCount int
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

var localeDirNames = map[string]bool{
	"locales": true, "locale": true, "i18n": true,
	"translations": true, "lang": true, "langs": true,
}

var srcExts = map[string]bool{
	".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".vue": true,
	".svelte": true, ".html": true, ".mjs": true, ".cjs": true,
}

var (
	// Explicit i18n calls: $t('key'), i18n.t('key'), translate('key')
	reExplicit = regexp.MustCompile(`(?:\$t|i18n\.t|translate)\s*\(\s*['"]([^'"]+)['"]`)
	// Bare t('key') — require non-identifier preceding char to avoid matching gettext(), next(), etc.
	reBareT = regexp.MustCompile(`(?:^|[^a-zA-Z_$\d])t\s*\(\s*['"]([^'"]+)['"]`)
	// Angular template pipe: 'key' | translate  or  'key' | transloco
	reAngularPipe = regexp.MustCompile(`['"]([^'"]+)['"]\s*\|\s*(?:translate|transloco)\b`)
	// Angular directive attribute: translate="key"  or  transloco="key"
	reAngularDirective = regexp.MustCompile(`(?:translate|transloco)="([^"]+)"`)
	// TranslateService / TranslocoService methods: .instant('key'), .stream('key'), .selectTranslate('key')
	reServiceMethod = regexp.MustCompile(`\.(?:instant|stream|selectTranslate)\s*\(\s*['"]([^'"]+)['"]`)
)

// Scan walks root, parses locale JSON files, scans source for t() calls,
// and returns which keys are unused and which references are undefined.
func Scan(root string) (*ScanResult, error) {
	// Pass 1: collect defined keys from locale files.
	defined := make(map[string]*pkg.I18nKey)
	var localeFiles []string

	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".json" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if !isLocaleFile(rel) {
			return nil
		}
		keys, err := parseLocaleFile(path, rel)
		if err != nil {
			return nil
		}
		localeFiles = append(localeFiles, rel)
		for key, k := range keys {
			if _, exists := defined[key]; !exists {
				defined[key] = k
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Pass 2: scan source files for i18n function calls.
	refs := make(map[string]*pkg.I18nKey)
	srcCount := 0

	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !srcExts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		srcCount++
		rel, _ := filepath.Rel(root, path)
		for key, k := range extractRefs(path, rel) {
			if _, exists := refs[key]; !exists {
				refs[key] = k
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Unused: defined in locale but never called in source.
	var unused []*pkg.I18nKey
	for key, k := range defined {
		if _, ok := refs[key]; !ok {
			unused = append(unused, k)
		}
	}
	sort.Slice(unused, func(i, j int) bool {
		if unused[i].File != unused[j].File {
			return unused[i].File < unused[j].File
		}
		return unused[i].Key < unused[j].Key
	})

	// Undefined: called in source but absent from all locale files.
	var undefined []*pkg.I18nKey
	for key, k := range refs {
		if _, ok := defined[key]; !ok {
			undefined = append(undefined, k)
		}
	}
	sort.Slice(undefined, func(i, j int) bool {
		if undefined[i].File != undefined[j].File {
			return undefined[i].File < undefined[j].File
		}
		return undefined[i].Line < undefined[j].Line
	})

	return &ScanResult{
		Unused:       unused,
		Undefined:    undefined,
		LocaleFiles:  localeFiles,
		TotalDefined: len(defined),
		SrcFileCount: srcCount,
	}, nil
}

// isLocaleFile returns true if the relative path sits inside a known locale directory.
func isLocaleFile(relPath string) bool {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for _, part := range parts[:len(parts)-1] {
		if localeDirNames[strings.ToLower(part)] {
			return true
		}
	}
	return false
}

func parseLocaleFile(absPath, relPath string) (map[string]*pkg.I18nKey, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	result := make(map[string]*pkg.I18nKey)
	flattenKeys(obj, "", relPath, result)
	return result, nil
}

// flattenKeys recursively flattens a nested JSON object into dot-notation keys.
func flattenKeys(obj map[string]interface{}, prefix, file string, out map[string]*pkg.I18nKey) {
	for k, v := range obj {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch child := v.(type) {
		case string:
			out[key] = &pkg.I18nKey{Key: key, File: file}
		case map[string]interface{}:
			flattenKeys(child, key, file, out)
		}
	}
}

// extractRefs scans a source file line by line and returns all i18n key references found.
func extractRefs(absPath, relPath string) map[string]*pkg.I18nKey {
	refs := make(map[string]*pkg.I18nKey)

	f, err := os.Open(absPath)
	if err != nil {
		return refs
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		lb := []byte(sc.Text())

		for _, re := range []*regexp.Regexp{reExplicit, reBareT, reAngularPipe, reAngularDirective, reServiceMethod} {
			for _, m := range re.FindAllSubmatch(lb, -1) {
				key := string(m[1])
				if _, ok := refs[key]; !ok {
					refs[key] = &pkg.I18nKey{Key: key, File: relPath, Line: lineNum}
				}
			}
		}
	}
	return refs
}
