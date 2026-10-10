package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MasterOfBinary/griffin/internal/strategy"
)

func TestRunIDDependsOnModeVersionAndDataset(t *testing.T) {
	base, err := RunID("demo", "1", "abc")
	if err != nil {
		t.Fatal(err)
	}
	same, err := RunID("demo", "1", "abc")
	if err != nil {
		t.Fatal(err)
	}
	if base != same {
		t.Fatalf("id not stable: %s vs %s", base, same)
	}
	for _, alt := range []struct{ mode, ver, hash string }{
		{"signal_only", "1", "abc"},
		{"strict", "1", "abc"},
		{"demo", "2", "abc"},
		{"demo", "1", "abd"},
	} {
		id, err := RunID(alt.mode, alt.ver, alt.hash)
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
	id, err := RunID("demo", "1", "dataset")
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
	id, err := RunID("demo", "1", "dataset")
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
