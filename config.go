package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const configRelativePath = ".zot/autoresearch.json"

type config struct {
	Objective      string   `json:"objective"`
	Benchmark      string   `json:"benchmark"`
	ScorePattern   string   `json:"score_pattern"`
	Direction      string   `json:"direction"`
	Unit           string   `json:"unit"`
	EditablePaths  []string `json:"editable_paths"`
	MaxIterations  int      `json:"max_iterations"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	MinDelta       float64  `json:"min_delta"`
}

func defaultConfig() config {
	return config{
		Objective:      "Improve the benchmark score without changing its semantics.",
		Benchmark:      "go test ./... -run '^$' -bench . -count 1",
		ScorePattern:   `(?m)^score:\s*([-+]?[0-9]*\.?[0-9]+)`,
		Direction:      "minimize",
		Unit:           "score",
		EditablePaths:  []string{"."},
		MaxIterations:  20,
		TimeoutSeconds: 55,
		MinDelta:       0,
	}
}

func (a *app) configPath() string {
	return filepath.Join(a.cwd, filepath.FromSlash(configRelativePath))
}

func (a *app) loadConfig() (config, error) {
	data, err := os.ReadFile(a.configPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config{}, fmt.Errorf("configuration not found; run /autoresearch init")
		}
		return config{}, fmt.Errorf("read configuration: %w", err)
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("parse configuration: %w", err)
	}
	if err := validateConfig(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg config) error {
	if cfg.Objective == "" {
		return errors.New("objective is required")
	}
	if cfg.Benchmark == "" {
		return errors.New("benchmark is required")
	}
	if cfg.Direction != "minimize" && cfg.Direction != "maximize" {
		return errors.New(`direction must be "minimize" or "maximize"`)
	}
	if len(cfg.EditablePaths) == 0 {
		return errors.New("editable_paths must contain at least one project-relative path")
	}
	for _, path := range cfg.EditablePaths {
		clean := filepath.Clean(path)
		if path == "" || filepath.IsAbs(path) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("editable path %q must stay inside the project", path)
		}
	}
	if cfg.MaxIterations <= 0 {
		return errors.New("max_iterations must be greater than zero")
	}
	if cfg.TimeoutSeconds <= 0 || cfg.TimeoutSeconds > 55 {
		return errors.New("timeout_seconds must be between 1 and 55 (zot tools time out after 60 seconds)")
	}
	re, err := regexp.Compile(cfg.ScorePattern)
	if err != nil {
		return fmt.Errorf("invalid score_pattern: %w", err)
	}
	if re.NumSubexp() < 1 {
		return errors.New("score_pattern must contain a capture group for the numeric score")
	}
	return nil
}

func validateEditablePaths(cwd string, paths []string) error {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	root = filepath.Clean(root)
	for _, path := range paths {
		candidate := filepath.Join(root, path)
		resolved, err := filepath.EvalSymlinks(candidate)
		if errors.Is(err, os.ErrNotExist) {
			resolved = candidate
		} else if err != nil {
			return fmt.Errorf("resolve editable path %q: %w", path, err)
		}
		resolved, err = filepath.Abs(resolved)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("editable path %q resolves outside the project", path)
		}
	}
	return nil
}

func (a *app) initConfig() error {
	path := a.configPath()
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", configRelativePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
