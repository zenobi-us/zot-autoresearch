package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const configRelativePath = ".zot/autoresearch.json"

var configRelativePaths = []string{
	configRelativePath,
	".zot/autoresearch.toml",
	".zot/autoresearch.yaml",
	".zot/autoresearch.yml",
}

type config struct {
	Objective      string   `json:"objective" toml:"objective" yaml:"objective"`
	Benchmark      string   `json:"benchmark" toml:"benchmark" yaml:"benchmark"`
	ScorePattern   string   `json:"score_pattern" toml:"score_pattern" yaml:"score_pattern"`
	Direction      string   `json:"direction" toml:"direction" yaml:"direction"`
	Unit           string   `json:"unit" toml:"unit" yaml:"unit"`
	EditablePaths  []string `json:"editable_paths" toml:"editable_paths" yaml:"editable_paths"`
	MaxIterations  int      `json:"max_iterations" toml:"max_iterations" yaml:"max_iterations"`
	TimeoutSeconds int      `json:"timeout_seconds" toml:"timeout_seconds" yaml:"timeout_seconds"`
	MinDelta       float64  `json:"min_delta" toml:"min_delta" yaml:"min_delta"`
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

func (a *app) configPath(relativePath string) string {
	return filepath.Join(a.cwd, filepath.FromSlash(relativePath))
}

func (a *app) existingConfigPaths() ([]string, error) {
	var found []string
	for _, relativePath := range configRelativePaths {
		_, err := os.Stat(a.configPath(relativePath))
		switch {
		case err == nil:
			found = append(found, relativePath)
		case errors.Is(err, os.ErrNotExist):
			continue
		default:
			return nil, fmt.Errorf("inspect configuration %s: %w", relativePath, err)
		}
	}
	return found, nil
}

func (a *app) loadConfig() (config, error) {
	paths, err := a.existingConfigPaths()
	if err != nil {
		return config{}, err
	}
	if len(paths) == 0 {
		return config{}, fmt.Errorf("configuration not found; create one of %s (or run /autoresearch init)", strings.Join(configRelativePaths, ", "))
	}
	if len(paths) > 1 {
		return config{}, fmt.Errorf("multiple configuration files found (%s); keep only one", strings.Join(paths, ", "))
	}

	path := paths[0]
	data, err := os.ReadFile(a.configPath(path))
	if err != nil {
		return config{}, fmt.Errorf("read configuration %s: %w", path, err)
	}
	cfg, err := parseConfig(path, data)
	if err != nil {
		return config{}, fmt.Errorf("parse configuration %s: %w", path, err)
	}
	if err := validateConfig(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func parseConfig(path string, data []byte) (config, error) {
	var cfg config
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		err = json.Unmarshal(data, &cfg)
	case ".toml":
		err = toml.Unmarshal(data, &cfg)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &cfg)
	default:
		return config{}, fmt.Errorf("unsupported configuration format %q", filepath.Ext(path))
	}
	return cfg, err
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

func (a *app) initConfig(goal, format string) (string, error) {
	paths, err := a.existingConfigPaths()
	if err != nil {
		return "", err
	}
	if len(paths) > 0 {
		return "", fmt.Errorf("configuration already exists: %s", strings.Join(paths, ", "))
	}

	relativePath, err := configPathForFormat(format)
	if err != nil {
		return "", err
	}
	cfg := defaultConfig()
	if goal != "" {
		cfg.Objective = goal
	}
	data, err := marshalConfig(format, cfg)
	if err != nil {
		return "", err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	path := a.configPath(relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return relativePath, nil
}

func configPathForFormat(format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		return ".zot/autoresearch.json", nil
	case "toml":
		return ".zot/autoresearch.toml", nil
	case "yaml":
		return ".zot/autoresearch.yaml", nil
	case "yml":
		return ".zot/autoresearch.yml", nil
	default:
		return "", fmt.Errorf("unsupported format %q; use json, toml, yaml, or yml", format)
	}
}

func marshalConfig(format string, cfg config) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(cfg, "", "  ")
	case "toml":
		return toml.Marshal(cfg)
	case "yaml", "yml":
		return yaml.Marshal(cfg)
	default:
		return nil, fmt.Errorf("unsupported format %q; use json, toml, yaml, or yml", format)
	}
}
