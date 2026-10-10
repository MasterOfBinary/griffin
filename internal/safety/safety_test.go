package safety

import (
	"strings"
	"testing"

	"github.com/MasterOfBinary/griffin/internal/candles"
)

func TestParseMode(t *testing.T) {
	for _, name := range []string{"demo", "signal_only", "strict"} {
		mode, err := ParseMode(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(mode) != name {
			t.Fatalf("mode %s", mode)
		}
	}
	if _, err := ParseMode("live"); err == nil {
		t.Fatal("expected live to be rejected")
	}
}

func TestListenLoopbackOnly(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8787", "127.0.0.1:0", "[::1]:8787"} {
		if err := ValidateListen(addr); err != nil {
			t.Fatalf("%s: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:8787", "localhost:8787", "192.168.1.10:8787", "127.0.0.1", ":8787"} {
		if err := ValidateListen(addr); err == nil {
			t.Fatalf("expected %s to be refused", addr)
		}
	}
}

func TestStrictSpread(t *testing.T) {
	bar := candles.Bar{Symbol: "SOL", SpreadBps: 80, LiquidityUSD: 5_000_000}
	err := CheckBar(bar, Limits{MaxSpreadBps: 25, MinLiquidityUSD: 1_000_000})
	if err == nil {
		t.Fatal("expected spread failure")
	}
}

func TestRedactKeyMaterial(t *testing.T) {
	secret := "abcdEFGHijklMNOPqrstUVWX1234567890"
	raw := "api" + "_key=" + secret
	if !ContainsKeyMaterial(raw) {
		t.Fatal("expected detector to match")
	}
	out := Redact(raw)
	if strings.Contains(out, secret) {
		t.Fatalf("secret leaked: %s", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatalf("redacted output %q", out)
	}
}
