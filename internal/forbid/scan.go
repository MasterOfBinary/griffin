// Package forbid fails the build when the repository grows a forbidden path.
package forbid

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Violation is one forbidden path match.
type Violation struct {
	Path    string
	Line    int
	Pattern string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: %s", v.Path, v.Line, v.Pattern)
}

// Scan walks root and returns every forbidden path match outside the denylist file.
func Scan(root string) ([]Violation, error) {
	patterns := Patterns()
	lower := make([]string, len(patterns))
	for i, pat := range patterns {
		lower[i] = strings.ToLower(pat)
	}
	banned := map[string]bool{}
	for _, name := range BannedNames() {
		banned[name] = true
	}
	var out []Violation
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "bin", "var", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		slash := filepath.ToSlash(path)
		if strings.HasSuffix(slash, "internal/forbid/denylist.go") {
			return nil
		}
		if banned[d.Name()] {
			out = append(out, Violation{Path: path, Line: 1, Pattern: d.Name()})
			return nil
		}
		if !textFile(path) {
			return nil
		}
		matches, err := scanFile(path, lower)
		if err != nil {
			return err
		}
		out = append(out, matches...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func textFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".md", ".json", ".mod", ".sum", ".yml", ".yaml", ".txt":
		return true
	}
	switch filepath.Base(path) {
	case "Makefile", ".gitignore":
		return true
	default:
		return false
	}
}

func scanFile(path string, patterns []string) ([]Violation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var out []Violation
	lineNo := 1
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if strings.ContainsRune(line, '\x00') {
				return nil, nil
			}
			folded := strings.ToLower(line)
			for i, pat := range patterns {
				if strings.Contains(folded, pat) {
					out = append(out, Violation{Path: path, Line: lineNo, Pattern: Patterns()[i]})
				}
			}
			lineNo++
		}
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}
}
