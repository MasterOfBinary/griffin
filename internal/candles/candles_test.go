package candles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidateSynthetic(t *testing.T) {
	ds, err := LoadFile(filepath.Join(repoRoot(t), "fixtures", "synthetic", "majors.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ds.Name != "synthetic-majors" {
		t.Fatalf("name %q", ds.Name)
	}
	if len(ds.Hash) != 64 {
		t.Fatalf("hash %q", ds.Hash)
	}
	bars := Filter(ds.Bars, []string{"ETH", "BTC", "SOL"})
	if err := Validate(bars); err != nil {
		t.Fatal(err)
	}
	if err := Covers(bars, []string{"SOL", "BTC", "ETH"}); err != nil {
		t.Fatal(err)
	}
}

func TestHashChangesWhenBytesChange(t *testing.T) {
	a := HashBytes([]byte("alpha"))
	b := HashBytes([]byte("beta"))
	if a == b {
		t.Fatal("expected different hashes")
	}
}

func TestRejectsBadOHLC(t *testing.T) {
	err := Validate([]Bar{{
		Symbol: "SOL", Ts: 1, Open: 10, High: 9, Low: 8, Close: 10, Volume: 1,
	}})
	if err == nil {
		t.Fatal("expected ohlc error")
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
