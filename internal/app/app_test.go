package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MasterOfBinary/griffin/internal/candles"
	"github.com/MasterOfBinary/griffin/internal/store"
)

func TestHelpListsModes(t *testing.T) {
	var buf bytes.Buffer
	if err := Execute(context.Background(), []string{"help"}, &buf); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"demo", "signal_only", "strict"} {
		if !strings.Contains(buf.String(), mode) {
			t.Fatalf("help missing %s\n%s", mode, buf.String())
		}
	}
}

func TestCLIModesAndFinishedRerun(t *testing.T) {
	dir := t.TempDir()
	demoOut := mustRun(t, dir, "demo")
	if !strings.Contains(demoOut, "mode: demo\n") {
		t.Fatalf("demo output:\n%s", demoOut)
	}
	if strings.Contains(demoOut, "fills: 0\n") {
		t.Fatalf("demo recorded no fills:\n%s", demoOut)
	}
	if !strings.Contains(demoOut, "serve: skipped\n") {
		t.Fatalf("demo output:\n%s", demoOut)
	}

	signalOut := mustRun(t, dir, "run", "--mode", "signal_only")
	if !strings.Contains(signalOut, "mode: signal_only\n") || !strings.Contains(signalOut, "fills: 0\n") {
		t.Fatalf("signal_only output:\n%s", signalOut)
	}

	strictOut := mustRun(t, dir, "run", "--mode", "strict")
	if !strings.Contains(strictOut, "mode: strict\n") || strings.Contains(strictOut, "fills: 0\n") || strings.Contains(strictOut, "halted: true\n") {
		t.Fatalf("strict output:\n%s", strictOut)
	}

	err := Execute(context.Background(), cmd(t, dir, "demo"), &bytes.Buffer{})
	if !errors.Is(err, store.ErrRunFinished) {
		t.Fatalf("expected finished rerun to be refused, got %v", err)
	}

	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	runs := st.List()
	if len(runs) != 3 {
		t.Fatalf("stored %d runs", len(runs))
	}
	seen := map[string]bool{}
	for _, run := range runs {
		seen[run.Mode] = true
		if run.Status != store.StatusFinished {
			t.Fatalf("status %s", run.Status)
		}
	}
	for _, mode := range []string{"demo", "signal_only", "strict"} {
		if !seen[mode] {
			t.Fatalf("missing stored mode %s", mode)
		}
	}
}

func TestStrictHaltRefusesRerun(t *testing.T) {
	root := repoRoot(t)
	cfg, err := LoadConfig(filepath.Join(root, "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Mode = "strict"
	cfg.DataDir = t.TempDir()
	cfg.Dataset = wideSpreadFixture(t, filepath.Join(root, "fixtures", "synthetic", "majors.json"))
	run, err := RunOnce(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !run.Halted || len(run.Fills) != 0 || run.Status != store.StatusFinished {
		t.Fatalf("run %+v", run)
	}
	if run.Mode != "strict" {
		t.Fatalf("mode %s", run.Mode)
	}
	_, err = RunOnce(cfg)
	if !errors.Is(err, store.ErrRunFinished) {
		t.Fatalf("got %v", err)
	}
	cfg.ConfigVersion = "2"
	again, err := RunOnce(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == run.ID {
		t.Fatal("config version did not change the run id")
	}
}

func TestListenRefusalDoesNotStore(t *testing.T) {
	dir := t.TempDir()
	err := Execute(context.Background(), append(cmd(t, dir, "demo"), "--listen", "0.0.0.0:8787"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "runs.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("store stat: %v", statErr)
	}
}

func TestDemoServesLoopback(t *testing.T) {
	dir := t.TempDir()
	addr := freeLoopback(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var buf bytes.Buffer
	errCh := make(chan error, 1)
	args := []string{"demo", "--config", configPath(t), "--data-dir", dir, "--listen", addr}
	go func() {
		errCh <- Execute(ctx, args, &buf)
	}()

	deadline := time.Now().Add(3 * time.Second)
	var body string
	var last error
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			raw, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			body = string(raw)
			break
		}
		last = err
		time.Sleep(20 * time.Millisecond)
	}
	if body == "" {
		t.Fatal(last)
	}
	cancel()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "127.0.0.1") {
		t.Fatalf("page missing loopback address:\n%s", body)
	}
	for _, mode := range []string{"demo", "signal_only", "strict"} {
		if !strings.Contains(body, ">"+mode+"<") {
			t.Fatalf("page missing %s\n%s", mode, body)
		}
	}
	out := buf.String()
	if !strings.Contains(out, "mode: demo\n") || !strings.Contains(out, "listening: http://"+addr+"\n") {
		t.Fatalf("cli output:\n%s", out)
	}
}

func mustRun(t *testing.T, dir string, command ...string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Execute(context.Background(), cmd(t, dir, command...), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func cmd(t *testing.T, dir string, command ...string) []string {
	t.Helper()
	args := append([]string{}, command...)
	return append(args, "--config", configPath(t), "--data-dir", dir, "--once")
}

func configPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "config.example.json")
}

func wideSpreadFixture(t *testing.T, src string) string {
	t.Helper()
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Name string        `json:"name"`
		Bars []candles.Bar `json:"bars"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Bars) == 0 {
		t.Fatal("empty fixture")
	}
	doc.Bars[0].SpreadBps = 400
	out := filepath.Join(t.TempDir(), "wide.json")
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

func freeLoopback(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
