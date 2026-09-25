package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

func TestParseInitArgs(t *testing.T) {
	goal, format, err := parseInitArgs("reduce parser allocations TOML")
	if err != nil {
		t.Fatal(err)
	}
	if goal != "reduce parser allocations" || format != "toml" {
		t.Fatalf("parseInitArgs = %q, %q", goal, format)
	}

	goal, format, err = parseInitArgs("")
	if err != nil || goal != "" || format != "json" {
		t.Fatalf("default parseInitArgs = %q, %q, %v", goal, format, err)
	}

	if _, _, err := parseInitArgs("goal xml"); err == nil {
		t.Fatal("parseInitArgs accepted an unsupported format")
	}
}

func TestLoadConfigSupportsJSONTOMLAndYAML(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
	}{
		{
			name: "JSON",
			path: ".zot/autoresearch.json",
			content: `{
  "objective": "lower latency",
  "benchmark": "./bench.sh",
  "score_pattern": "score: ([0-9.]+)",
  "direction": "minimize",
  "unit": "ms",
  "editable_paths": ["src", "go.mod"],
  "max_iterations": 12,
  "timeout_seconds": 30,
  "min_delta": 0.5
}`,
		},
		{
			name: "TOML",
			path: ".zot/autoresearch.toml",
			content: `objective = "lower latency"
benchmark = "./bench.sh"
score_pattern = 'score: ([0-9.]+)'
direction = "minimize"
unit = "ms"
editable_paths = ["src", "go.mod"]
max_iterations = 12
timeout_seconds = 30
min_delta = 0.5
`,
		},
		{
			name: "YAML",
			path: ".zot/autoresearch.yaml",
			content: `objective: lower latency
benchmark: ./bench.sh
score_pattern: 'score: ([0-9.]+)'
direction: minimize
unit: ms
editable_paths:
  - src
  - go.mod
max_iterations: 12
timeout_seconds: 30
min_delta: 0.5
`,
		},
		{
			name: "YML",
			path: ".zot/autoresearch.yml",
			content: `objective: lower latency
benchmark: ./bench.sh
score_pattern: 'score: ([0-9.]+)'
direction: minimize
unit: ms
editable_paths: [src, go.mod]
max_iterations: 12
timeout_seconds: 30
min_delta: 0.5
`,
		},
	}

	want := config{
		Objective: "lower latency", Benchmark: "./bench.sh", ScorePattern: `score: ([0-9.]+)`,
		Direction: "minimize", Unit: "ms", EditablePaths: []string{"src", "go.mod"},
		MaxIterations: 12, TimeoutSeconds: 30, MinDelta: 0.5,
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, filepath.FromSlash(test.path)), test.content)
			a := newApp(ext.New(extensionName, version))
			a.cwd = dir

			got, err := a.loadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("config = %#v, want %#v", got, want)
			}
		})
	}
}

func TestLoadConfigReportsFormatParseErrors(t *testing.T) {
	tests := []struct {
		path    string
		content string
	}{
		{path: ".zot/autoresearch.json", content: "{"},
		{path: ".zot/autoresearch.toml", content: "objective = ["},
		{path: ".zot/autoresearch.yaml", content: "objective: ["},
	}
	for _, test := range tests {
		t.Run(filepath.Ext(test.path), func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, filepath.FromSlash(test.path)), test.content)
			a := newApp(ext.New(extensionName, version))
			a.cwd = dir

			_, err := a.loadConfig()
			if err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("loadConfig error = %v; want path %q", err, test.path)
			}
		})
	}
}

func TestLoadConfigRejectsMultipleFormats(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".zot", "autoresearch.json"), "{}")
	writeFile(t, filepath.Join(dir, ".zot", "autoresearch.toml"), "")
	a := newApp(ext.New(extensionName, version))
	a.cwd = dir

	_, err := a.loadConfig()
	if err == nil || !strings.Contains(err.Error(), "multiple configuration files found") {
		t.Fatalf("loadConfig error = %v", err)
	}
}

func TestInitConfigDoesNotOverwriteAlternateFormat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".zot", "autoresearch.yaml"), "objective: keep me\n")
	a := newApp(ext.New(extensionName, version))
	a.cwd = dir

	_, err := a.initConfig("new goal", "json")
	if err == nil || !strings.Contains(err.Error(), ".zot/autoresearch.yaml") {
		t.Fatalf("initConfig error = %v", err)
	}
}

func TestInitConfigWritesGoalInRequestedFormat(t *testing.T) {
	tests := []struct {
		format string
		path   string
	}{
		{format: "json", path: ".zot/autoresearch.json"},
		{format: "toml", path: ".zot/autoresearch.toml"},
		{format: "yaml", path: ".zot/autoresearch.yaml"},
		{format: "yml", path: ".zot/autoresearch.yml"},
	}
	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			a := newApp(ext.New(extensionName, version))
			a.cwd = t.TempDir()

			path, err := a.initConfig("reduce parser allocations", test.format)
			if err != nil {
				t.Fatal(err)
			}
			if path != test.path {
				t.Fatalf("path = %q, want %q", path, test.path)
			}
			cfg, err := a.loadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Objective != "reduce parser allocations" {
				t.Fatalf("objective = %q", cfg.Objective)
			}
		})
	}
}

func TestScopedPathsExcludeEveryConfigFormat(t *testing.T) {
	got := scopedPaths([]string{"."})
	for _, path := range configRelativePaths {
		want := ":(exclude)" + path
		if !containsString(got, want) {
			t.Fatalf("scoped paths %q do not contain %q", got, want)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
