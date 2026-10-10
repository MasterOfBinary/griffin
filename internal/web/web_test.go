package web

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MasterOfBinary/griffin/internal/sim"
	"github.com/MasterOfBinary/griffin/internal/store"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

func TestPageShowsModesAndRun(t *testing.T) {
	st := seed(t)
	h := New(st, "127.0.0.1:8787")
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := get(t, srv.URL+"/")
	for _, mode := range []string{"demo", "signal_only", "strict"} {
		if !strings.Contains(body, ">"+mode+"<") {
			t.Fatalf("index missing labelled mode %s\n%s", mode, body)
		}
	}
	if !strings.Contains(body, "127.0.0.1:8787") {
		t.Fatalf("index missing bind address\n%s", body)
	}
	id, err := store.RunID("demo", "1", "hash", "fp")
	if err != nil {
		t.Fatal(err)
	}
	detail := get(t, srv.URL+"/runs/"+id)
	if !strings.Contains(detail, `class="active">demo`) {
		t.Fatalf("detail missing active demo label\n%s", detail)
	}
	if !strings.Contains(detail, "fast average crossed above slow average") {
		t.Fatal("detail missing signal")
	}
	if !strings.Contains(detail, "No paper fills in this run.") {
		t.Fatal("expected the empty fill state for this seeded demo row")
	}
	strictID, err := store.RunID("strict", "1", "hash", "fp")
	if err != nil {
		t.Fatal(err)
	}
	strictPage := get(t, srv.URL+"/runs/"+strictID)
	if !strings.Contains(strictPage, "<td>buy</td>") {
		t.Fatalf("strict page missing paper fill\n%s", strictPage)
	}
}

func TestServeRefusesNonLoopback(t *testing.T) {
	_, err := Listen("0.0.0.0:8787")
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("got %v", err)
	}
}

func TestListenReportsBoundPort(t *testing.T) {
	ln, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	bound := ln.Addr().String()
	if strings.HasSuffix(bound, ":0") || !strings.HasPrefix(bound, "127.0.0.1:") {
		t.Fatalf("bound address %s", bound)
	}
}

func TestRejectsRebindingHost(t *testing.T) {
	h := New(emptyStore(t), "127.0.0.1:8787")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestReloadsRunsWrittenAfterOpen(t *testing.T) {
	dir := t.TempDir()
	st := emptyStoreIn(t, dir)
	h := New(st, "127.0.0.1:8787")
	later, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.RunID("demo", "1", "hash", "fp")
	if err != nil {
		t.Fatal(err)
	}
	if err := later.Finish(store.Run{
		ID: id, Mode: "demo", ConfigVersion: "1", DatasetHash: "hash",
		Status: store.StatusFinished, StrategyName: "school-griffin-1",
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), id) {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestServeLoopback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr := freeLoopback(t)
	ln, err := Listen(addr)
	if err != nil {
		t.Fatal(err)
	}
	bound := ln.Addr().String()
	h := New(emptyStore(t), bound)
	errCh := make(chan error, 1)
	go func() { errCh <- Serve(ctx, ln, h) }()

	deadline := time.Now().Add(2 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + bound + "/healthz")
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !strings.Contains(string(body), "listen="+bound) {
				t.Fatalf("healthz %s", body)
			}
			cancel()
			select {
			case err := <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("server did not stop")
			}
			return
		}
		last = err
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(last)
}

func seed(t *testing.T) *store.Store {
	t.Helper()
	st := emptyStore(t)
	for _, mode := range []string{"demo", "signal_only", "strict"} {
		id, err := store.RunID(mode, "1", "hash", "fp")
		if err != nil {
			t.Fatal(err)
		}
		run := store.Run{
			ID: id, Mode: mode, ConfigVersion: "1", DatasetHash: "hash",
			DatasetName: "synthetic-majors", StrategyName: "school-griffin-1",
			Status: store.StatusFinished, StartingCash: 1000, Cash: 1000, Equity: 1000,
			Signals: []strategy.Signal{{
				Symbol: "SOL",
				Action: strategy.ActionEnterLong,
				Reason: "fast average crossed above slow average",
				Fast:   1,
				Slow:   2,
			}},
		}
		if mode == "strict" {
			run.Fills = []sim.Fill{{Symbol: "SOL", Side: "buy", Qty: 1, Price: 100}}
		}
		if err := st.Finish(run); err != nil {
			t.Fatal(err)
		}
	}
	return st
}

func emptyStore(t *testing.T) *store.Store {
	t.Helper()
	return emptyStoreIn(t, t.TempDir())
}

func emptyStoreIn(t *testing.T, dir string) *store.Store {
	t.Helper()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func get(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	return string(body)
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
