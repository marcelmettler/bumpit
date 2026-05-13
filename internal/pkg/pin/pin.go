package pin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

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

var sections = []struct {
	jsonKey string
	depType pkg.DepType
}{
	{"dependencies", pkg.DepDependencies},
	{"devDependencies", pkg.DepDevDependencies},
	{"peerDependencies", pkg.DepPeerDependencies},
	{"optionalDependencies", pkg.DepOptionalDependencies},
}

// Scan walks root for package.json files and returns all dependencies
// whose version spec starts with ^ or ~.
func Scan(root string) ([]*pkg.UnpinnedDep, error) {
	var result []*pkg.UnpinnedDep

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		dir := filepath.Dir(path)
		rel, _ := filepath.Rel(root, path)
		deps, err := scanFile(path, rel, dir, dirLabel(dir, root))
		if err != nil {
			return nil
		}
		result = append(result, deps...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Section != b.Section {
			return a.Section < b.Section
		}
		return a.Name < b.Name
	})
	return result, nil
}

func scanFile(absPath, relPath, dir, dirName string) ([]*pkg.UnpinnedDep, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var result []*pkg.UnpinnedDep
	for _, sec := range sections {
		if raw[sec.jsonKey] == nil {
			continue
		}
		var deps map[string]string
		if err := json.Unmarshal(raw[sec.jsonKey], &deps); err != nil {
			continue
		}
		for name, version := range deps {
			if !strings.HasPrefix(version, "^") && !strings.HasPrefix(version, "~") {
				continue
			}
			result = append(result, &pkg.UnpinnedDep{
				Name:    name,
				Version: version,
				Pinned:  version[1:],
				File:    relPath,
				Dir:     dir,
				DirName: dirName,
				Section: sec.jsonKey,
				DepType: sec.depType,
			})
		}
	}
	return result, nil
}

// Pin writes exact version strings into the package.json files for the
// selected deps, stripping their ^ or ~ prefix in-place without reformatting.
// Returns the number of deps successfully pinned.
func Pin(root string, deps []*pkg.UnpinnedDep) (int, error) {
	byFile := make(map[string][]*pkg.UnpinnedDep)
	for _, d := range deps {
		byFile[d.File] = append(byFile[d.File], d)
	}

	count := 0
	for relPath, fileDeps := range byFile {
		absPath := filepath.Join(root, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
			return count, fmt.Errorf("%s: %w", relPath, err)
		}
		for _, d := range fileDeps {
			// Match exact JSON key-value pair to avoid touching unrelated fields.
			old := fmt.Sprintf(`%q: %q`, d.Name, d.Version)
			replacement := fmt.Sprintf(`%q: %q`, d.Name, d.Pinned)
			data = bytes.ReplaceAll(data, []byte(old), []byte(replacement))
		}
		if err := os.WriteFile(absPath, data, 0644); err != nil {
			return count, fmt.Errorf("%s: %w", relPath, err)
		}
		count += len(fileDeps)
	}
	return count, nil
}

func dirLabel(dir, root string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "root"
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return parts[len(parts)-1]
}
