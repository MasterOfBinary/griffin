package market

import "testing"

func TestUniverse(t *testing.T) {
	if err := ValidateUniverse(QuoteUSDC, []string{"SOL", "BTC", "ETH"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateUniverse(QuoteUSDC, []string{"DOGE"}); err == nil {
		t.Fatal("expected long-tail symbol to be rejected")
	}
	if err := ValidateUniverse("USD", []string{"SOL"}); err == nil {
		t.Fatal("expected unsupported quote to be rejected")
	}
}
