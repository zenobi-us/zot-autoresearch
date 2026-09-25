package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type runState struct {
	Active        bool        `json:"active"`
	Config        config      `json:"config"`
	Objective     string      `json:"objective"`
	Direction     string      `json:"direction"`
	Unit          string      `json:"unit"`
	Baseline      *float64    `json:"baseline,omitempty"`
	Best          *float64    `json:"best,omitempty"`
	BestCommit    string      `json:"best_commit,omitempty"`
	StartedAt     time.Time   `json:"started_at,omitempty"`
	StoppedAt     time.Time   `json:"stopped_at,omitempty"`
	MaxIterations int         `json:"max_iterations"`
	Iterations    []iteration `json:"iterations,omitempty"`
}

type iteration struct {
	Number     int       `json:"number"`
	Status     string    `json:"status"`
	Score      float64   `json:"score"`
	Gain       float64   `json:"gain"`
	Hypothesis string    `json:"hypothesis"`
	Summary    string    `json:"summary"`
	Commit     string    `json:"commit,omitempty"`
	Output     string    `json:"output,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (a *app) statePath() (string, error) {
	if a.dataDir == "" {
		return "", errors.New("host did not provide an extension data directory")
	}
	sum := sha256.Sum256([]byte(filepath.Clean(a.cwd)))
	return filepath.Join(a.dataDir, "projects", hex.EncodeToString(sum[:8])+".json"), nil
}

func (a *app) loadState() error {
	path, err := a.statePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var state runState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	a.mu.Lock()
	a.state = state
	a.mu.Unlock()
	return nil
}

func (a *app) saveStateLocked() error {
	path, err := a.statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}
