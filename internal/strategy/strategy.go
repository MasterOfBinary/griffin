// Package strategy implements school-griffin-1, a paper signal on liquid majors.
// Signals are research output. Nothing in this package sends an order.
package strategy

import (
	"fmt"
	"math"

	"github.com/MasterOfBinary/griffin/internal/candles"
)

// Action values stored on a run.
const (
	ActionEnterLong = "enter_long"
	ActionExit      = "exit"
	ActionHold      = "hold"
)

// Params configures the dual moving-average cross.
type Params struct {
	Name string
	Fast int
	Slow int
}

// Signal is a state change for one symbol at one bar close.
type Signal struct {
	Symbol string  `json:"symbol"`
	Ts     int64   `json:"ts"`
	Action string  `json:"action"`
	Reason string  `json:"reason"`
	Fast   float64 `json:"fast"`
	Slow   float64 `json:"slow"`
}

// Validate checks period ordering.
func (p Params) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("strategy name is empty")
	}
	if p.Fast < 2 {
		return fmt.Errorf("fast period must be at least 2")
	}
	if p.Slow <= p.Fast {
		return fmt.Errorf("slow period must be greater than fast period")
	}
	return nil
}

// Run emits enter and exit signals when the fast average crosses the slow average.
// Bars between crosses are holds and are not recorded.
func Run(bars []candles.Bar, p Params) ([]Signal, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	bySymbol := map[string][]candles.Bar{}
	var order []string
	seen := map[string]bool{}
	for _, bar := range bars {
		bySymbol[bar.Symbol] = append(bySymbol[bar.Symbol], bar)
		if !seen[bar.Symbol] {
			order = append(order, bar.Symbol)
			seen[bar.Symbol] = true
		}
	}
	var out []Signal
	for _, sym := range order {
		signals, err := runSymbol(bySymbol[sym], p)
		if err != nil {
			return nil, err
		}
		out = append(out, signals...)
	}
	return out, nil
}

func runSymbol(bars []candles.Bar, p Params) ([]Signal, error) {
	if len(bars) < p.Slow {
		return nil, fmt.Errorf("%s has %d bars; slow period is %d", bars[0].Symbol, len(bars), p.Slow)
	}
	closes := make([]float64, len(bars))
	for i, bar := range bars {
		closes[i] = bar.Close
	}
	fast := ema(closes, p.Fast)
	slow := ema(closes, p.Slow)
	var out []Signal
	var prev float64
	ready := false
	for i := range bars {
		if math.IsNaN(fast[i]) || math.IsNaN(slow[i]) {
			continue
		}
		diff := fast[i] - slow[i]
		if !ready {
			prev = diff
			ready = true
			continue
		}
		switch {
		case prev <= 0 && diff > 0:
			out = append(out, Signal{
				Symbol: bars[i].Symbol,
				Ts:     bars[i].Ts,
				Action: ActionEnterLong,
				Reason: "fast average crossed above slow average",
				Fast:   fast[i],
				Slow:   slow[i],
			})
		case prev >= 0 && diff < 0:
			out = append(out, Signal{
				Symbol: bars[i].Symbol,
				Ts:     bars[i].Ts,
				Action: ActionExit,
				Reason: "fast average crossed below slow average",
				Fast:   fast[i],
				Slow:   slow[i],
			})
		}
		prev = diff
	}
	return out, nil
}

// ema returns a series seeded with the simple average of the first period.
// Values before the seed are NaN.
func ema(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	for i := range out {
		out[i] = math.NaN()
	}
	if len(values) < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	out[period-1] = sum / float64(period)
	k := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		out[i] = values[i]*k + out[i-1]*(1-k)
	}
	return out
}
