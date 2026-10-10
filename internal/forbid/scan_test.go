package forbid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryHasNoForbiddenPaths(t *testing.T) {
	violations, err := Scan(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		var b stringsBuilder
		for _, v := range violations {
			b.WriteString(v.String())
			b.WriteByte('\n')
		}
		t.Fatalf("forbidden paths found:\n%s", b.String())
	}
}

func TestScanReportsForbiddenPath(t *testing.T) {
	dir := t.TempDir()
	token := Patterns()[0]
	path := filepath.Join(dir, "bad.go")
	body := "package bad\nconst Marker = \"" + token + "\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a violation")
	}
}

func TestScanReportsBannedFilename(t *testing.T) {
	dir := t.TempDir()
	name := BannedNames()[0]
	if err := os.WriteFile(filepath.Join(dir, name), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a banned filename")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

type stringsBuilder struct {
	buf []byte
}

func (b *stringsBuilder) WriteString(s string) {
	b.buf = append(b.buf, s...)
}

func (b *stringsBuilder) WriteByte(c byte) {
	b.buf = append(b.buf, c)
}

func (b *stringsBuilder) String() string {
	return string(b.buf)
}
