package app

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/MasterOfBinary/griffin/internal/market"
	"github.com/MasterOfBinary/griffin/internal/safety"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

// Config is the versioned paper-lab configuration.
// config_version is part of the run id: bump it when the settings change.
type Config struct {
	ConfigVersion string         `json:"config_version"`
	Mode          string         `json:"mode"`
	Listen        string         `json:"listen"`
	DataDir       string         `json:"data_dir"`
	Quote         string         `json:"quote"`
	Universe      []string       `json:"universe"`
	Dataset       string         `json:"dataset"`
	StartingCash  float64        `json:"starting_cash"`
	FeeBps        float64        `json:"fee_bps"`
	SlippageBps   float64        `json:"slippage_bps"`
	Strategy      StrategyConfig `json:"strategy"`
	Safety        safety.Limits  `json:"safety"`
}

// StrategyConfig is the school-griffin-1 parameter block.
type StrategyConfig struct {
	Name           string  `json:"name"`
	Fast           int     `json:"fast"`
	Slow           int     `json:"slow"`
	MaxPositionPct float64 `json:"max_position_pct"`
}

var versionPattern = regexp.MustCompile(`^[a-z0-9._-]{1,32}$`)

// LoadConfig reads a JSON config and validates it.
func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if safety.ContainsKeyMaterial(string(raw)) {
		return Config{}, fmt.Errorf("config contains key material")
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks the closed universe, the listen address, and the strategy parameters.
func (c Config) Validate() error {
	if !versionPattern.MatchString(c.ConfigVersion) {
		return fmt.Errorf("config_version %q must be a short lowercase token", c.ConfigVersion)
	}
	if _, err := safety.ParseMode(c.Mode); err != nil {
		return err
	}
	if err := safety.ValidateListen(c.Listen); err != nil {
		return err
	}
	if c.DataDir == "" {
		return fmt.Errorf("data_dir is empty")
	}
	if c.Dataset == "" {
		return fmt.Errorf("dataset is empty")
	}
	if err := market.ValidateUniverse(c.Quote, c.Universe); err != nil {
		return err
	}
	if c.StartingCash <= 0 {
		return fmt.Errorf("starting_cash must be positive")
	}
	if c.FeeBps < 0 || c.SlippageBps < 0 {
		return fmt.Errorf("fees and slippage cannot be negative")
	}
	if c.Strategy.MaxPositionPct <= 0 || c.Strategy.MaxPositionPct > 1 {
		return fmt.Errorf("max_position_pct must be in (0, 1]")
	}
	if err := (strategy.Params{Name: c.Strategy.Name, Fast: c.Strategy.Fast, Slow: c.Strategy.Slow}).Validate(); err != nil {
		return err
	}
	if c.Safety.MaxNotional <= 0 || c.Safety.MaxSpreadBps <= 0 || c.Safety.MinLiquidityUSD <= 0 {
		return fmt.Errorf("safety limits must be positive")
	}
	return nil
}
