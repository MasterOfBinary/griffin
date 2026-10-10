package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MasterOfBinary/griffin/internal/strategy"
)

func TestRunIDDependsOnModeVersionAndDataset(t *testing.T) {
	base, err := RunID("demo", "1", "abc", "fp")
	if err != nil {
		t.Fatal(err)
	}
	same, err := RunID("demo", "1", "abc", "fp")
	if err != nil {
		t.Fatal(err)
	}
	if base != same {
		t.Fatalf("id not stable: %s vs %s", base, same)
	}
	for _, alt := range []struct{ mode, ver, hash, fp string }{
		{"signal_only", "1", "abc", "fp"},
		{"strict", "1", "abc", "fp"},
		{"demo", "2", "abc", "fp"},
		{"demo", "1", "abd", "fp"},
		{"demo", "1", "abc", "other"},
	} {
		id, err := RunID(alt.mode, alt.ver, alt.hash, alt.fp)
		if err != nil {
			t.Fatal(err)
		}
		if id == base {
			t.Fatalf("expected a new id for %+v", alt)
		}
	}
}

func TestRefuseFinishedRerun(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, err := RunID("demo", "1", "dataset", "fp")
	if err != nil {
		t.Fatal(err)
	}
	run := Run{
		ID: id, Mode: "demo", ConfigVersion: "1", DatasetHash: "dataset",
		Status: StatusFinished, Signals: []strategy.Signal{}, Fills: nil,
	}
	if err := st.Finish(run); err != nil {
		t.Fatal(err)
	}
	err = st.Finish(run)
	if !errors.Is(err, ErrRunFinished) {
		t.Fatalf("got %v", err)
	}
	reopen, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopen.Get(id)
	if !ok || got.Mode != "demo" || got.Status != StatusFinished {
		t.Fatalf("reloaded %+v ok=%v", got, ok)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs.json")); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRejectsKeyMaterial(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, err := RunID("demo", "1", "dataset", "fp")
	if err != nil {
		t.Fatal(err)
	}
	run := Run{
		ID: id, Mode: "demo", ConfigVersion: "1", DatasetHash: "dataset",
		Status: StatusFinished,
		Signals: []strategy.Signal{{
			Symbol: "SOL", Action: strategy.ActionHold, Reason: "api" + "_key=" + strings.Repeat("ab", 16),
		}},
	}
	if err := st.Finish(run); err == nil {
		t.Fatal("expected key material to be refused")
	}
	if _, ok := st.Get(id); ok {
		t.Fatal("run was stored")
	}
}

func TestConcurrentFinishKeepsBothRuns(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, mode := range []string{"demo", "strict"} {
		wg.Add(1)
		go func(mode string) {
			defer wg.Done()
			st, err := Open(dir)
			if err != nil {
				errCh <- err
				return
			}
			id, err := RunID(mode, "1", "dataset", "fp")
			if err != nil {
				errCh <- err
				return
			}
			errCh <- st.Finish(Run{
				ID: id, Mode: mode, ConfigVersion: "1", DatasetHash: "dataset",
				Status: StatusFinished,
			})
		}(mode)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.List()) != 2 {
		t.Fatalf("stored %d runs", len(st.List()))
	}
}
