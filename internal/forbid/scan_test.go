package forbid

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MasterOfBinary/griffin/internal/repotest"
)

func TestRepositoryHasNoForbiddenPaths(t *testing.T) {
	violations, err := Scan(repotest.Root(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		var b strings.Builder
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
	for _, name := range []string{
		BannedNames()[0],
		"Dockerfile.dev",
		"compose.yaml",
		filepath.Join("vendor", "docker-compose.yml"),
		filepath.Join("bin", "compose.yml"),
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("FROM scratch\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	violations, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) < 5 {
		t.Fatalf("got %d violations, want at least 5: %v", len(violations), violations)
	}
}

func TestScanReadsScriptsAndKeepsEarlierNULMatch(t *testing.T) {
	dir := t.TempDir()
	token := Patterns()[0]
	script := filepath.Join(dir, "bin", "note.py")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "print('ok')\nmarker = '" + token + "'\n\x00tail\nmarker2 = '" + token + "'\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("TOKEN="+token+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	var sawPy, sawEnv bool
	for _, v := range violations {
		if strings.HasSuffix(v.Path, "note.py") {
			sawPy = true
		}
		if strings.HasSuffix(v.Path, ".env") {
			sawEnv = true
		}
	}
	if !sawPy || !sawEnv {
		t.Fatalf("violations: %v", violations)
	}
}
