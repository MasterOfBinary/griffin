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
	bar := candles.Bar{Symbol: "SOL", SpreadBps: 80, SpreadKnown: true, LiquidityUSD: 5_000_000}
	err := CheckBar(bar, Limits{MaxSpreadBps: 25, MinLiquidityUSD: 1_000_000})
	if err == nil {
		t.Fatal("expected spread failure")
	}
	missing := candles.Bar{Symbol: "SOL", SpreadBps: 0, LiquidityUSD: 5_000_000}
	if err := CheckBar(missing, Limits{MaxSpreadBps: 25, MinLiquidityUSD: 1_000_000}); err == nil {
		t.Fatal("expected missing spread to fail")
	}
	negative := candles.Bar{Symbol: "SOL", SpreadBps: -1, SpreadKnown: true, LiquidityUSD: 5_000_000}
	if err := CheckBar(negative, Limits{MaxSpreadBps: 25, MinLiquidityUSD: 1_000_000}); err == nil {
		t.Fatal("expected negative spread to fail")
	}
	zero := candles.Bar{Symbol: "SOL", SpreadBps: 0, SpreadKnown: true, LiquidityUSD: 5_000_000}
	if err := CheckBar(zero, Limits{MaxSpreadBps: 25, MinLiquidityUSD: 1_000_000}); err != nil {
		t.Fatal(err)
	}
}

func TestRedactKeyMaterial(t *testing.T) {
	secret := "abcdEFGHijklMNOPqrstUVWX1234567890"
	plain := `{"config_version":"1","mode":"demo","fee_bps":1}`
	if ContainsKeyMaterial(plain) || Redact(plain) != plain {
		t.Fatalf("plain config was treated as key material: %s", Redact(plain))
	}
	samples := []string{
		"api" + "_key=" + secret,
		`{"` + "private" + "_key" + `": "` + secret + ` extra"}`,
		"SECRET" + "_KEY=" + secret,
		"client_secret=" + secret,
		"AWS_" + "SECRET" + "_ACCESS_KEY=" + secret,
		"-----BEGIN " + "PRIVATE" + " KEY-----\n" + secret + "\n-----END " + "PRIVATE" + " KEY-----",
		"mnemonic: " + secret + " secondword thirdword",
	}
	for _, raw := range samples {
		if !ContainsKeyMaterial(raw) {
			t.Fatalf("expected detector to match %q", raw)
		}
		out := Redact(raw)
		if strings.Contains(out, secret) || strings.Contains(out, "secondword") || strings.Contains(out, "thirdword") {
			t.Fatalf("secret leaked from %q into %q", raw, out)
		}
		if !strings.Contains(out, "[redacted]") {
			t.Fatalf("redacted output %q", out)
		}
	}
}
