package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

func TestParseScoreAndComparison(t *testing.T) {
	score, err := parseScore("setup\nscore: 12.75 ms\ndone", `(?m)^score:\s*([0-9.]+)`)
	if err != nil {
		t.Fatal(err)
	}
	if score != 12.75 {
		t.Fatalf("score = %v", score)
	}
	if !improved(9, 10, "minimize", 0.5) || improved(9.6, 10, "minimize", 0.5) {
		t.Fatal("minimize threshold comparison is incorrect")
	}
	if !improved(11, 10, "maximize", 0.5) || improved(10.4, 10, "maximize", 0.5) {
		t.Fatal("maximize threshold comparison is incorrect")
	}
	if got := gainPercent(8, 10, "minimize"); math.Abs(got-20) > 0.001 {
		t.Fatalf("gain = %v", got)
	}
}

func TestCommandRouterHelpAndUnknown(t *testing.T) {
	a := newApp(ext.New(extensionName, version))
	if got := a.command(""); got.Action != "display" || !strings.Contains(got.Display, "/autoresearch start") {
		t.Fatalf("help response = %#v", got)
	}
	if got := a.command("wat"); got.Error == "" || !strings.Contains(got.Error, "unknown autoresearch subcommand") {
		t.Fatalf("unknown response = %#v", got)
	}
}

func TestExperimentAcceptsImprovementAndRestoresRegression(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "data")
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Autoresearch Test")
	runGit(t, dir, "config", "user.email", "autoresearch@example.invalid")

	cfg := config{
		Objective: "lower score", Benchmark: "cat score.txt", ScorePattern: `score:\s*([0-9.]+)`,
		Direction: "minimize", Unit: "ms", EditablePaths: []string{"score.txt"},
		MaxIterations: 3, TimeoutSeconds: 5,
	}
	writeConfig(t, dir, cfg)
	writeFile(t, filepath.Join(dir, "score.txt"), "score: 10\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	a := newApp(ext.New(extensionName, version))
	a.cwd = dir
	a.dataDir = dataDir
	a.state = runState{Active: true, Config: cfg, Objective: cfg.Objective, Direction: cfg.Direction, Unit: cfg.Unit, MaxIterations: cfg.MaxIterations}

	message, err := a.runExperiment(experimentArgs{Hypothesis: "baseline", Summary: "establish baseline"})
	if err != nil || !strings.Contains(message, "Baseline established") {
		t.Fatalf("baseline: message=%q err=%v", message, err)
	}

	writeFile(t, filepath.Join(dir, "score.txt"), "score: 8\n")
	message, err = a.runExperiment(experimentArgs{Hypothesis: "make it faster", Summary: "reduce score"})
	if err != nil || !strings.Contains(message, "ACCEPTED #1") {
		t.Fatalf("accept: message=%q err=%v", message, err)
	}
	if a.state.Best == nil || *a.state.Best != 8 || a.state.BestCommit == "" {
		t.Fatalf("accepted state = %#v", a.state)
	}

	writeFile(t, filepath.Join(dir, "score.txt"), "score: 12\n")
	message, err = a.runExperiment(experimentArgs{Hypothesis: "bad idea", Summary: "increase score"})
	if err != nil || !strings.Contains(message, "REJECTED #2") {
		t.Fatalf("reject: message=%q err=%v", message, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "score.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "score: 8\n" {
		t.Fatalf("restored score.txt = %q", data)
	}
	if dirty, err := dirtyPaths(dir, nil); err != nil || dirty != "" {
		t.Fatalf("working tree after reject = %q, %v", dirty, err)
	}
}

func TestPanelShowsTotalsAndCommit(t *testing.T) {
	baseline, best := 100.0, 80.0
	state := runState{
		Active: true, Objective: "go faster", Direction: "minimize", Unit: "ms",
		Baseline: &baseline, Best: &best, BestCommit: "abc1234", MaxIterations: 10,
		Iterations: []iteration{
			{Status: "baseline", Score: 100, Hypothesis: "baseline"},
			{Number: 1, Status: "accepted", Score: 80, Gain: 20, Commit: "abc1234", Hypothesis: "cache values"},
			{Number: 2, Status: "rejected", Score: 90, Gain: 10, Hypothesis: "extra layer"},
		},
	}
	title, lines, _ := panelForState(state)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(title, "running") || !strings.Contains(joined, "1 accepted") || !strings.Contains(joined, "abc1234") || !strings.Contains(joined, "rejected") {
		t.Fatalf("panel:\n%s\n%s", title, joined)
	}
}

func writeConfig(t *testing.T, dir string, cfg config) {
	t.Helper()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, configRelativePath), string(data))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if output, err := git(dir, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
