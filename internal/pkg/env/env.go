package env

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/marcelmettler/chorekit/internal/pkg"
)

// ScanResult holds the outcome of an environment variable audit.
type ScanResult struct {
	Unused       []*pkg.EnvVar // defined in .env.example but never referenced in source
	Undefined    []*pkg.EnvVar // referenced in source but absent from all .env.example files
	EnvFiles     []string      // .env.example files that were scanned
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

// envFileNames are the committed, secret-free env definition files we scan for declarations.
var envFileNames = map[string]bool{
	".env.example":  true,
	".env.sample":   true,
	".env.template": true,
	".env.defaults": true,
	".env.schema":   true,
}

var srcExts = map[string]bool{
	".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".vue": true,
	".svelte": true, ".mjs": true, ".cjs": true,
	".go": true, ".py": true, ".sh": true, ".bash": true,
	".yaml": true, ".yml": true,
}

// reEnvLine parses a line from a .env file: optional "export ", then KEY=...
var reEnvLine = regexp.MustCompile(`^(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=`)

// refPatterns detect env var access in source code.
var refPatterns = []*regexp.Regexp{
	// Node.js: process.env.KEY or process.env['KEY']
	regexp.MustCompile(`process\.env\.([A-Z_][A-Z0-9_]*)`),
	regexp.MustCompile(`process\.env\[['"]([^'"]+)['"]\]`),
	// Vite / Astro: import.meta.env.KEY or import.meta.env['KEY']
	regexp.MustCompile(`import\.meta\.env\.([A-Z_][A-Z0-9_]*)`),
	regexp.MustCompile(`import\.meta\.env\[['"]([^'"]+)['"]\]`),
	// Go: os.Getenv("KEY")
	regexp.MustCompile(`os\.Getenv\(['"]([^'"]+)['"]\)`),
	// Python: os.environ["KEY"] or os.environ.get("KEY")
	regexp.MustCompile(`os\.environ\[['"]([^'"]+)['"]\]`),
	regexp.MustCompile(`os\.environ\.get\(['"]([^'"]+)['"]`),
	// YAML / shell: ${KEY} substitution (docker-compose, k8s, shell scripts)
	regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`),
}

// Scan walks root, reads .env.example files for declared vars, scans source for
// references, and returns which vars are unused and which are undefined.
func Scan(root string) (*ScanResult, error) {
	// Pass 1: collect declared vars from .env.example files.
	defined := make(map[string]*pkg.EnvVar)
	var envFiles []string

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
		if !envFileNames[d.Name()] {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		envFiles = append(envFiles, rel)
		vars, err := parseEnvFile(path, rel)
		if err != nil {
			return nil
		}
		for key, v := range vars {
			if _, exists := defined[key]; !exists {
				defined[key] = v
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Pass 2: scan source files for env var references.
	refs := make(map[string]*pkg.EnvVar)
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
		for key, v := range extractRefs(path, rel) {
			if _, exists := refs[key]; !exists {
				refs[key] = v
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Unused: declared but never referenced.
	var unused []*pkg.EnvVar
	for key, v := range defined {
		if _, ok := refs[key]; !ok {
			unused = append(unused, v)
		}
	}
	sort.Slice(unused, func(i, j int) bool {
		if unused[i].File != unused[j].File {
			return unused[i].File < unused[j].File
		}
		return unused[i].Key < unused[j].Key
	})

	// Undefined: referenced in source but absent from all .env.example files.
	var undefined []*pkg.EnvVar
	for key, v := range refs {
		if _, ok := defined[key]; !ok {
			undefined = append(undefined, v)
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
		EnvFiles:     envFiles,
		TotalDefined: len(defined),
		SrcFileCount: srcCount,
	}, nil
}

func parseEnvFile(absPath, relPath string) (map[string]*pkg.EnvVar, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vars := make(map[string]*pkg.EnvVar)
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := reEnvLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := m[1]
		vars[key] = &pkg.EnvVar{Key: key, File: relPath, Line: lineNum}
	}
	return vars, sc.Err()
}

func extractRefs(absPath, relPath string) map[string]*pkg.EnvVar {
	refs := make(map[string]*pkg.EnvVar)

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
		for _, re := range refPatterns {
			for _, m := range re.FindAllSubmatch(lb, -1) {
				key := string(m[1])
				if _, ok := refs[key]; !ok {
					refs[key] = &pkg.EnvVar{Key: key, File: relPath, Line: lineNum}
				}
			}
		}
	}
	return refs
}
