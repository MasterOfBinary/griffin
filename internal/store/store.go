// Package store persists finished paper runs as JSON files. There is no database server.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"

	"github.com/MasterOfBinary/griffin/internal/safety"
	"github.com/MasterOfBinary/griffin/internal/sim"
	"github.com/MasterOfBinary/griffin/internal/strategy"
)

// ErrRunFinished is returned when a finished run id is submitted again.
var ErrRunFinished = errors.New("run is finished; choose a new run id by changing mode, configuration, or dataset")

// StatusFinished is the only terminal status. A finished id cannot be reused.
const StatusFinished = "finished"

// Run is the stored record for one labelled mode and dataset.
type Run struct {
	ID            string            `json:"id"`
	Mode          string            `json:"mode"`
	ConfigVersion string            `json:"config_version"`
	DatasetHash   string            `json:"dataset_hash"`
	DatasetName   string            `json:"dataset_name"`
	StrategyName  string            `json:"strategy_name"`
	Status        string            `json:"status"`
	Halted        bool              `json:"halted"`
	HaltReason    string            `json:"halt_reason,omitempty"`
	StartingCash  float64           `json:"starting_cash"`
	Cash          float64           `json:"cash"`
	Equity        float64           `json:"equity"`
	Listen        string            `json:"listen,omitempty"`
	Signals       []strategy.Signal `json:"signals"`
	Fills         []sim.Fill        `json:"fills"`
	Positions     []sim.Position    `json:"positions"`
	FinishedAt    string            `json:"finished_at"`
}

type fileDoc struct {
	Runs []Run `json:"runs"`
}

// Store is a process-local JSON document under a data directory.
type Store struct {
	mu   sync.Mutex
	path string
	runs map[string]Run
}

// Open loads dir/runs.json, creating the directory if needed.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("data dir is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, "runs.json")
	s := &Store{path: path, runs: map[string]Run{}}
	if err := s.reloadFromDisk(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload re-reads runs.json so a long-lived UI sees runs written after startup.
func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadFromDisk()
}

// Get returns a run by id.
func (s *Store) Get(id string) (Run, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[id]
	return run, ok
}

// List returns runs ordered by id.
func (s *Store) List() []Run {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Run, 0, len(s.runs))
	for _, run := range s.runs {
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Finish inserts a finished run. A second finish of the same id is refused.
func (s *Store) Finish(run Run) error {
	if run.ID == "" {
		return fmt.Errorf("run id is empty")
	}
	if run.Status != StatusFinished {
		return fmt.Errorf("only finished runs are stored")
	}
	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if safety.ContainsKeyMaterial(string(raw)) {
		return fmt.Errorf("refusing to store key material")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withFileLock(func() error {
		if err := s.reloadFromDisk(); err != nil {
			return err
		}
		if prev, ok := s.runs[run.ID]; ok && prev.Status == StatusFinished {
			return fmt.Errorf("%w: %s", ErrRunFinished, run.ID)
		}
		s.runs[run.ID] = run
		return s.writeLocked()
	})
}

func (s *Store) withFileLock(fn func() error) error {
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open store lock: %w", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock store: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func (s *Store) reloadFromDisk() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.runs = map[string]Run{}
			return nil
		}
		return fmt.Errorf("read store: %w", err)
	}
	runs, err := parseRuns(raw)
	if err != nil {
		return err
	}
	s.runs = runs
	return nil
}

func parseRuns(raw []byte) (map[string]Run, error) {
	if safety.ContainsKeyMaterial(string(raw)) {
		return nil, fmt.Errorf("store file contains key material and will not be loaded")
	}
	var doc fileDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse store: %w", err)
	}
	runs := make(map[string]Run, len(doc.Runs))
	for _, run := range doc.Runs {
		runs[run.ID] = run
	}
	return runs, nil
}

func (s *Store) writeLocked() error {
	doc := fileDoc{Runs: make([]Run, 0, len(s.runs))}
	for _, run := range s.runs {
		doc.Runs = append(doc.Runs, run)
	}
	sort.Slice(doc.Runs, func(i, j int) bool { return doc.Runs[i].ID < doc.Runs[j].ID })
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if safety.ContainsKeyMaterial(string(raw)) {
		return fmt.Errorf("refusing to store key material")
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".runs-*.json")
	if err != nil {
		return fmt.Errorf("create store temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close store temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("commit store: %w", err)
	}
	cleanup = false
	return nil
}
