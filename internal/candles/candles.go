// Package candles loads OHLCV bars from local fixtures. It does not fetch remote markets.
package candles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Bar is one OHLCV observation for a symbol.
type Bar struct {
	Symbol       string  `json:"symbol"`
	Ts           int64   `json:"ts"`
	Open         float64 `json:"open"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
	Close        float64 `json:"close"`
	Volume       float64 `json:"volume"`
	SpreadBps    float64 `json:"spread_bps"`
	LiquidityUSD float64 `json:"liquidity_usd"`
}

// Dataset is a named collection of bars plus the hash of the bytes that were loaded.
type Dataset struct {
	Name string
	Hash string
	Bars []Bar
}

type fileDoc struct {
	Name string `json:"name"`
	Bars []Bar  `json:"bars"`
}

// LoadFile reads one dataset file and hashes its raw bytes.
func LoadFile(path string) (Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, fmt.Errorf("read dataset: %w", err)
	}
	var doc fileDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Dataset{}, fmt.Errorf("parse dataset: %w", err)
	}
	if doc.Name == "" {
		doc.Name = path
	}
	return Dataset{Name: doc.Name, Hash: hashBytes(raw), Bars: doc.Bars}, nil
}

// HashBytes returns the hex SHA-256 of b. Exported for tests that mutate fixtures.
func HashBytes(b []byte) string {
	return hashBytes(b)
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Filter returns bars whose symbol is in universe, ordered by time then symbol.
func Filter(bars []Bar, universe []string) []Bar {
	allow := map[string]bool{}
	for _, sym := range universe {
		allow[sym] = true
	}
	out := make([]Bar, 0, len(bars))
	for _, bar := range bars {
		if allow[bar.Symbol] {
			out = append(out, bar)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ts == out[j].Ts {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].Ts < out[j].Ts
	})
	return out
}

// Validate checks OHLC consistency and per-symbol monotonic timestamps.
func Validate(bars []Bar) error {
	if len(bars) == 0 {
		return fmt.Errorf("dataset has no bars")
	}
	last := map[string]int64{}
	seen := map[string]int{}
	for _, bar := range bars {
		if bar.Symbol == "" {
			return fmt.Errorf("bar missing symbol")
		}
		if bar.Ts <= 0 {
			return fmt.Errorf("%s has a non-positive timestamp", bar.Symbol)
		}
		if prev, ok := last[bar.Symbol]; ok && bar.Ts <= prev {
			return fmt.Errorf("%s timestamps are not strictly increasing", bar.Symbol)
		}
		last[bar.Symbol] = bar.Ts
		seen[bar.Symbol]++
		if err := validateOHLC(bar); err != nil {
			return err
		}
	}
	return nil
}

func validateOHLC(bar Bar) error {
	if bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 || bar.Close <= 0 {
		return fmt.Errorf("%s has a non-positive price at %s", bar.Symbol, time.UnixMilli(bar.Ts).UTC().Format(time.RFC3339))
	}
	if bar.Low > bar.High {
		return fmt.Errorf("%s low above high", bar.Symbol)
	}
	if bar.High < bar.Open || bar.High < bar.Close {
		return fmt.Errorf("%s high below open or close", bar.Symbol)
	}
	if bar.Low > bar.Open || bar.Low > bar.Close {
		return fmt.Errorf("%s low above open or close", bar.Symbol)
	}
	if bar.Volume < 0 {
		return fmt.Errorf("%s has negative volume", bar.Symbol)
	}
	return nil
}

// Covers reports symbols in universe that have no bars.
func Covers(bars []Bar, universe []string) error {
	got := map[string]bool{}
	for _, bar := range bars {
		got[bar.Symbol] = true
	}
	for _, sym := range universe {
		if !got[sym] {
			return fmt.Errorf("dataset is missing %s", sym)
		}
	}
	return nil
}
