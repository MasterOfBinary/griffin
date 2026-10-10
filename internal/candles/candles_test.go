package candles

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/MasterOfBinary/griffin/internal/repotest"
)

func TestLoadAndValidateSynthetic(t *testing.T) {
	ds, err := LoadFile(filepath.Join(repotest.Root(t), "fixtures", "synthetic", "majors.json"))
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
	if !ds.Bars[0].SpreadKnown {
		t.Fatal("spread_bps was present but not recorded")
	}
}

func TestMissingSpreadStaysUnknown(t *testing.T) {
	var missing Bar
	if err := json.Unmarshal([]byte(`{"symbol":"SOL","ts":1,"open":1,"high":1,"low":1,"close":1,"volume":1}`), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.SpreadKnown {
		t.Fatal("missing spread_bps was treated as known")
	}
	var zero Bar
	if err := json.Unmarshal([]byte(`{"symbol":"SOL","ts":1,"open":1,"high":1,"low":1,"close":1,"volume":1,"spread_bps":0}`), &zero); err != nil {
		t.Fatal(err)
	}
	if !zero.SpreadKnown || zero.SpreadBps != 0 {
		t.Fatalf("%+v", zero)
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
