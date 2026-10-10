package strategy

import (
	"testing"

	"github.com/MasterOfBinary/griffin/internal/candles"
)

func TestCrossEmitsEnterThenExit(t *testing.T) {
	bars := syntheticPath("SOL")
	signals, err := Run(bars, Params{Name: "school-griffin-1", Fast: 8, Slow: 21})
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) < 2 {
		t.Fatalf("signals: %+v", signals)
	}
	if signals[0].Action != ActionEnterLong {
		t.Fatalf("first action %s", signals[0].Action)
	}
	sawExit := false
	for _, sig := range signals {
		if sig.Action == ActionExit {
			sawExit = true
		}
		if sig.Action == ActionHold {
			t.Fatal("holds are not stored")
		}
	}
	if !sawExit {
		t.Fatal("expected an exit")
	}
}

func TestRejectsPeriodOrder(t *testing.T) {
	_, err := Run(nil, Params{Name: "school-griffin-1", Fast: 21, Slow: 8})
	if err == nil {
		t.Fatal("expected parameter error")
	}
}

func syntheticPath(symbol string) []candles.Bar {
	bars := make([]candles.Bar, 60)
	var prev float64 = 100
	for i := range bars {
		var close float64
		switch {
		case i < 30:
			close = 100
		case i < 45:
			close = 100 + float64(i-29)*2
		default:
			close = 130 - float64(i-44)*3
		}
		op := prev
		bars[i] = candles.Bar{
			Symbol: symbol,
			Ts:     int64(i+1) * 3600000,
			Open:   op,
			High:   max(op, close) + 0.5,
			Low:    min(op, close) - 0.5,
			Close:  close,
			Volume: 1,
		}
		prev = close
	}
	return bars
}
