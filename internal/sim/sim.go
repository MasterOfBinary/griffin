// Package sim applies paper fills to signals. Fills stay inside the process.
package sim

import (
	"fmt"
	"sort"

	"github.com/MasterOfBinary/griffin/internal/candles"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

// Fill is a simulated paper fill at a later bar's open.
type Fill struct {
	Symbol string  `json:"symbol"`
	Ts     int64   `json:"ts"`
	Side   string  `json:"side"`
	Qty    float64 `json:"qty"`
	Price  float64 `json:"price"`
	Fee    float64 `json:"fee"`
	Reason string  `json:"reason"`
}

// Position is an open paper position.
type Position struct {
	Symbol   string  `json:"symbol"`
	Qty      float64 `json:"qty"`
	AvgPrice float64 `json:"avg_price"`
}

// Result is the book after walking the bars.
type Result struct {
	Cash      float64
	Equity    float64
	Fills     []Fill
	Positions []Position
}

// Config controls paper accounting. AllowFills is false in signal_only mode.
type Config struct {
	StartingCash   float64
	FeeBps         float64
	SlippageBps    float64
	MaxPositionPct float64
	MaxNotional    float64
	AllowFills     bool
}

// Paper walks bars in time order. A signal on one bar is eligible to fill on the next bar's open.
func Paper(bars []candles.Bar, signals []strategy.Signal, cfg Config) (Result, error) {
	if cfg.StartingCash <= 0 {
		return Result{}, fmt.Errorf("starting cash must be positive")
	}
	if cfg.MaxPositionPct <= 0 || cfg.MaxPositionPct > 1 {
		return Result{}, fmt.Errorf("max position pct must be in (0, 1]")
	}
	ordered := append([]candles.Bar(nil), bars...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Ts == ordered[j].Ts {
			return ordered[i].Symbol < ordered[j].Symbol
		}
		return ordered[i].Ts < ordered[j].Ts
	})
	sigAt := map[string]strategy.Signal{}
	for _, sig := range signals {
		sigAt[key(sig.Symbol, sig.Ts)] = sig
	}
	cash := cfg.StartingCash
	qty := map[string]float64{}
	avg := map[string]float64{}
	last := map[string]float64{}
	pending := map[string]string{}
	var fills []Fill
	for _, bar := range ordered {
		last[bar.Symbol] = bar.Close
		if cfg.AllowFills {
			if side, ok := pending[bar.Symbol]; ok {
				fill, nextCash, nextQty, nextAvg, err := apply(bar, side, cash, qty[bar.Symbol], avg[bar.Symbol], cfg)
				if err != nil {
					return Result{}, err
				}
				cash = nextCash
				qty[bar.Symbol] = nextQty
				avg[bar.Symbol] = nextAvg
				if fill.Qty > 0 {
					fills = append(fills, fill)
				}
				delete(pending, bar.Symbol)
			}
		}
		sig, ok := sigAt[key(bar.Symbol, bar.Ts)]
		if !ok {
			continue
		}
		switch sig.Action {
		case strategy.ActionEnterLong:
			if qty[bar.Symbol] == 0 {
				pending[bar.Symbol] = "buy"
			}
		case strategy.ActionExit:
			if qty[bar.Symbol] > 0 {
				pending[bar.Symbol] = "sell"
			}
		}
	}
	var positions []Position
	equity := cash
	for sym, q := range qty {
		if q <= 0 {
			continue
		}
		positions = append(positions, Position{Symbol: sym, Qty: q, AvgPrice: avg[sym]})
		equity += q * last[sym]
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].Symbol < positions[j].Symbol })
	return Result{Cash: cash, Equity: equity, Fills: fills, Positions: positions}, nil
}

func apply(bar candles.Bar, side string, cash, qty, avg float64, cfg Config) (Fill, float64, float64, float64, error) {
	switch side {
	case "buy":
		px := bar.Open * (1 + cfg.SlippageBps/10000)
		if cfg.FeeBps < 0 {
			return Fill{}, cash, qty, avg, fmt.Errorf("negative fee")
		}
		if px <= 0 || cash <= 0 {
			return Fill{}, cash, qty, avg, nil
		}
		// The fee is paid from the position budget, so pct=1 still fits in cash.
		feeRate := cfg.FeeBps / 10000
		budget := cash * cfg.MaxPositionPct
		if cfg.MaxNotional > 0 && budget > cfg.MaxNotional {
			budget = cfg.MaxNotional
		}
		if budget > cash {
			budget = cash
		}
		notional := budget / (1 + feeRate)
		fee := notional * feeRate
		if notional+fee > cash {
			notional = cash / (1 + feeRate)
			fee = cash - notional
		}
		if notional <= 0 {
			return Fill{}, cash, qty, avg, nil
		}
		q := notional / px
		return Fill{
			Symbol: bar.Symbol, Ts: bar.Ts, Side: "buy", Qty: q, Price: px, Fee: fee,
			Reason: "paper fill at next open",
		}, cash - notional - fee, q, px, nil
	case "sell":
		if qty <= 0 {
			return Fill{}, cash, qty, avg, nil
		}
		px := bar.Open * (1 - cfg.SlippageBps/10000)
		notional := qty * px
		fee := notional * cfg.FeeBps / 10000
		return Fill{
			Symbol: bar.Symbol, Ts: bar.Ts, Side: "sell", Qty: qty, Price: px, Fee: fee,
			Reason: "paper fill at next open",
		}, cash + notional - fee, 0, 0, nil
	default:
		return Fill{}, cash, qty, avg, fmt.Errorf("unknown paper side %q", side)
	}
}

func key(symbol string, ts int64) string {
	return fmt.Sprintf("%s:%d", symbol, ts)
}
