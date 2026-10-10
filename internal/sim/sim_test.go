package sim

import (
	"testing"

	"github.com/MasterOfBinary/griffin/internal/candles"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

func TestPaperFillsOnNextBar(t *testing.T) {
	bars := []candles.Bar{
		{Symbol: "SOL", Ts: 1, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1},
		{Symbol: "SOL", Ts: 2, Open: 110, High: 111, Low: 109, Close: 110, Volume: 1},
		{Symbol: "SOL", Ts: 3, Open: 90, High: 91, Low: 89, Close: 90, Volume: 1},
	}
	signals := []strategy.Signal{
		{Symbol: "SOL", Ts: 1, Action: strategy.ActionEnterLong},
		{Symbol: "SOL", Ts: 2, Action: strategy.ActionExit},
	}
	got, err := Paper(bars, signals, Config{
		StartingCash: 1000, FeeBps: 0, SlippageBps: 0, MaxPositionPct: 0.5, AllowFills: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fills) != 2 {
		t.Fatalf("fills %+v", got.Fills)
	}
	if got.Fills[0].Side != "buy" || got.Fills[0].Ts != 2 || got.Fills[0].Price != 110 {
		t.Fatalf("buy fill %+v", got.Fills[0])
	}
	if got.Fills[1].Side != "sell" || got.Fills[1].Ts != 3 || got.Fills[1].Price != 90 {
		t.Fatalf("sell fill %+v", got.Fills[1])
	}
	if len(got.Positions) != 0 {
		t.Fatalf("positions %+v", got.Positions)
	}
	if got.Cash >= 1000 {
		t.Fatalf("expected a loss on the round trip, cash %v", got.Cash)
	}
}

func TestSignalOnlySkipsFills(t *testing.T) {
	bars := []candles.Bar{
		{Symbol: "SOL", Ts: 1, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1},
		{Symbol: "SOL", Ts: 2, Open: 110, High: 111, Low: 109, Close: 110, Volume: 1},
	}
	got, err := Paper(bars, []strategy.Signal{
		{Symbol: "SOL", Ts: 1, Action: strategy.ActionEnterLong},
	}, Config{
		StartingCash: 1000, MaxPositionPct: 1, AllowFills: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Fills) != 0 {
		t.Fatalf("fills %+v", got.Fills)
	}
	if got.Cash != 1000 {
		t.Fatalf("cash %v", got.Cash)
	}
}
