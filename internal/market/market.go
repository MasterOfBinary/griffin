// Package market describes the liquid-major universe Griffin is allowed to study.
// Symbols outside this list are rejected. The lab does not grow a long-tail adapter.
package market

import "fmt"

// QuoteUSDC is the paper quote unit for every major in this lab.
const QuoteUSDC = "USDC"

// Instrument is a liquid major priced in the quote unit.
type Instrument struct {
	Symbol string
	Quote  string
}

// Majors is the closed universe. Changing it is a code change, not a config change.
var Majors = []Instrument{
	{Symbol: "SOL", Quote: QuoteUSDC},
	{Symbol: "BTC", Quote: QuoteUSDC},
	{Symbol: "ETH", Quote: QuoteUSDC},
}

// IsMajor reports whether symbol is in the closed universe.
func IsMajor(symbol string) bool {
	for _, inst := range Majors {
		if inst.Symbol == symbol {
			return true
		}
	}
	return false
}

// ValidateUniverse checks that every symbol is a known major and that quote matches.
func ValidateUniverse(quote string, symbols []string) error {
	if quote != QuoteUSDC {
		return fmt.Errorf("quote %q is not supported; this lab quotes %s", quote, QuoteUSDC)
	}
	if len(symbols) == 0 {
		return fmt.Errorf("universe is empty")
	}
	seen := map[string]bool{}
	for _, sym := range symbols {
		if !IsMajor(sym) {
			return fmt.Errorf("%s is not a liquid major in this lab", sym)
		}
		if seen[sym] {
			return fmt.Errorf("duplicate symbol %s", sym)
		}
		seen[sym] = true
	}
	return nil
}
