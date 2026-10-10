// Package safety labels run modes and rejects market states the strict mode will not paper-trade.
package safety

import (
	"fmt"

	"github.com/MasterOfBinary/griffin/internal/candles"
)

// Mode is the labelled operating mode stored with every run.
type Mode string

const (
	ModeDemo       Mode = "demo"
	ModeSignalOnly Mode = "signal_only"
	ModeStrict     Mode = "strict"
)

// Modes is the closed set shown by the CLI and the UI.
var Modes = []Mode{ModeDemo, ModeSignalOnly, ModeStrict}

// ParseMode accepts only the three labelled modes.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeDemo, ModeSignalOnly, ModeStrict:
		return Mode(s), nil
	default:
		return "", fmt.Errorf("unknown mode %q (want demo, signal_only, or strict)", s)
	}
}

// Limits are the market-quality gates used by strict mode.
type Limits struct {
	MaxNotional     float64 `json:"max_notional"`
	MaxSpreadBps    float64 `json:"max_spread_bps"`
	MinLiquidityUSD float64 `json:"min_liquidity_usd"`
}

// CheckBar reports whether a bar is eligible for a strict paper run.
func CheckBar(bar candles.Bar, lim Limits) error {
	if lim.MaxSpreadBps > 0 && bar.SpreadBps > lim.MaxSpreadBps {
		return fmt.Errorf("%s spread %.2f bps exceeds %.2f", bar.Symbol, bar.SpreadBps, lim.MaxSpreadBps)
	}
	if lim.MinLiquidityUSD > 0 && bar.LiquidityUSD < lim.MinLiquidityUSD {
		return fmt.Errorf("%s liquidity %.0f is below %.0f", bar.Symbol, bar.LiquidityUSD, lim.MinLiquidityUSD)
	}
	return nil
}

// CheckBook applies CheckBar to every bar. The first failure stops the book.
func CheckBook(bars []candles.Bar, lim Limits) error {
	for _, bar := range bars {
		if err := CheckBar(bar, lim); err != nil {
			return err
		}
	}
	return nil
}
