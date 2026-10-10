package app

import (
	"fmt"
	"time"

	"github.com/MasterOfBinary/griffin/internal/candles"
	"github.com/MasterOfBinary/griffin/internal/safety"
	"github.com/MasterOfBinary/griffin/internal/sim"
	"github.com/MasterOfBinary/griffin/internal/store"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

// RunOnce loads the dataset, derives the run id, and stores a finished paper run.
// A second call with the same mode, configuration version, and dataset is refused.
func RunOnce(cfg Config) (store.Run, error) {
	if err := cfg.Validate(); err != nil {
		return store.Run{}, err
	}
	mode, err := safety.ParseMode(cfg.Mode)
	if err != nil {
		return store.Run{}, err
	}
	ds, err := candles.LoadFile(cfg.Dataset)
	if err != nil {
		return store.Run{}, err
	}
	bars := candles.Filter(ds.Bars, cfg.Universe)
	if err := candles.Validate(bars); err != nil {
		return store.Run{}, err
	}
	if err := candles.Covers(bars, cfg.Universe); err != nil {
		return store.Run{}, err
	}
	id, err := store.RunID(string(mode), cfg.ConfigVersion, ds.Hash)
	if err != nil {
		return store.Run{}, err
	}
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return store.Run{}, err
	}
	if prev, ok := st.Get(id); ok && prev.Status == store.StatusFinished {
		return store.Run{}, fmt.Errorf("%w: %s", store.ErrRunFinished, id)
	}
	signals, err := strategy.Run(bars, strategy.Params{
		Name: cfg.Strategy.Name,
		Fast: cfg.Strategy.Fast,
		Slow: cfg.Strategy.Slow,
	})
	if err != nil {
		return store.Run{}, err
	}
	halted := false
	haltReason := ""
	if mode == safety.ModeStrict {
		if err := safety.CheckBook(bars, cfg.Safety); err != nil {
			halted = true
			haltReason = safety.Redact(err.Error())
		}
	}
	result, err := sim.Paper(bars, signals, sim.Config{
		StartingCash:   cfg.StartingCash,
		FeeBps:         cfg.FeeBps,
		SlippageBps:    cfg.SlippageBps,
		MaxPositionPct: cfg.Strategy.MaxPositionPct,
		MaxNotional:    cfg.Safety.MaxNotional,
		AllowFills:     mode != safety.ModeSignalOnly && !halted,
	})
	if err != nil {
		return store.Run{}, err
	}
	if signals == nil {
		signals = []strategy.Signal{}
	}
	if result.Fills == nil {
		result.Fills = []sim.Fill{}
	}
	if result.Positions == nil {
		result.Positions = []sim.Position{}
	}
	run := store.Run{
		ID:            id,
		Mode:          string(mode),
		ConfigVersion: cfg.ConfigVersion,
		DatasetHash:   ds.Hash,
		DatasetName:   ds.Name,
		StrategyName:  cfg.Strategy.Name,
		Status:        store.StatusFinished,
		Halted:        halted,
		HaltReason:    haltReason,
		StartingCash:  cfg.StartingCash,
		Cash:          result.Cash,
		Equity:        result.Equity,
		Listen:        cfg.Listen,
		Signals:       signals,
		Fills:         result.Fills,
		Positions:     result.Positions,
		FinishedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := st.Finish(run); err != nil {
		return store.Run{}, err
	}
	return run, nil
}

// Format renders a finished run for the CLI.
func Format(run store.Run) string {
	halt := "false"
	if run.Halted {
		halt = "true"
	}
	text := fmt.Sprintf(`mode: %s
run_id: %s
config_version: %s
dataset_hash: %s
dataset: %s
strategy: %s
signals: %d
fills: %d
halted: %s
cash: %.2f
equity: %.2f
`, run.Mode, run.ID, run.ConfigVersion, run.DatasetHash, run.DatasetName, run.StrategyName,
		len(run.Signals), len(run.Fills), halt, run.Cash, run.Equity)
	if run.Halted && run.HaltReason != "" {
		text += "halt_reason: " + safety.Redact(run.HaltReason) + "\n"
	}
	return text
}
