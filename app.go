package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

const panelID = "autoresearch-main"

type app struct {
	ext       *ext.Extension
	mu        sync.Mutex
	runMu     sync.Mutex
	cwd       string
	dataDir   string
	state     runState
	panelOpen bool
}

type experimentArgs struct {
	Hypothesis string `json:"hypothesis"`
	Summary    string `json:"summary"`
}

func newApp(e *ext.Extension) *app { return &app{ext: e} }

func (a *app) logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", extensionName, fmt.Sprintf(format, args...))
}

func (a *app) command(args string) ext.Response {
	fields := strings.Fields(args)
	subcommand := "help"
	if len(fields) > 0 {
		subcommand = strings.ToLower(fields[0])
	}
	rest := ""
	if len(fields) > 1 {
		rest = strings.TrimSpace(strings.Join(fields[1:], " "))
	}

	switch subcommand {
	case "help":
		return ext.Display(helpText())
	case "init":
		if err := a.initConfig(); err != nil {
			return ext.Errorf("autoresearch init: %v", err)
		}
		return ext.Display("Created " + configRelativePath + ". Edit the benchmark, score pattern, direction, and editable paths, then run /autoresearch start.")
	case "start":
		return a.start(rest)
	case "status":
		return ext.Display(a.statusText())
	case "watch", "history":
		a.setPanelOpen(true)
		title, lines, footer := a.panel()
		return ext.OpenPanel(panelID, title, lines, footer)
	case "stop":
		return a.stop()
	default:
		return ext.Errorf("unknown autoresearch subcommand %q; run /autoresearch help", subcommand)
	}
}

func helpText() string {
	return strings.TrimSpace(`Autoresearch commands

/autoresearch init             create .zot/autoresearch.json
/autoresearch start [objective] start a new experiment loop
/autoresearch watch            open the live iterations panel
/autoresearch status           show current run totals and best score
/autoresearch history          open the run history panel
/autoresearch stop             stop accepting experiments
/autoresearch help             show this help`)
}

func (a *app) start(objectiveOverride string) ext.Response {
	cfg, err := a.loadConfig()
	if err != nil {
		return ext.Errorf("autoresearch start: %v", err)
	}
	if objectiveOverride != "" {
		cfg.Objective = objectiveOverride
	}
	if err := ensureGitRepository(a.cwd); err != nil {
		return ext.Errorf("autoresearch start: %v", err)
	}
	if err := validateEditablePaths(a.cwd, cfg.EditablePaths); err != nil {
		return ext.Errorf("autoresearch start: %v", err)
	}
	if dirty, err := dirtyPaths(a.cwd, nil); err != nil {
		return ext.Errorf("autoresearch start: inspect working tree: %v", err)
	} else if dirty != "" {
		return ext.Errorf("autoresearch start refused: the working tree has pre-existing changes; commit or stash them first:\n%s", dirty)
	}

	now := time.Now().UTC()
	a.mu.Lock()
	a.state = runState{
		Active:        true,
		Config:        cfg,
		Objective:     cfg.Objective,
		Direction:     cfg.Direction,
		Unit:          cfg.Unit,
		StartedAt:     now,
		MaxIterations: cfg.MaxIterations,
	}
	err = a.saveStateLocked()
	a.mu.Unlock()
	if err != nil {
		return ext.Errorf("autoresearch start: save state: %v", err)
	}

	return ext.Prompt(researchPrompt(cfg))
}

func researchPrompt(cfg config) string {
	return fmt.Sprintf(`Run an autonomous autoresearch loop for this project.

Objective: %s
Benchmark: %s
Direction: %s
Editable paths: %s
Maximum candidate iterations: %d

Rules:
1. First call autoresearch_experiment without editing files to establish the baseline. Use hypothesis "baseline" and summary "establish baseline".
2. Then inspect the code and benchmark, form one focused hypothesis, and edit only the configured editable paths.
3. Call autoresearch_experiment after every candidate. It runs the benchmark, records the score, commits improvements, and restores rejected candidates.
4. Read the tool result before choosing the next experiment. Never commit, reset, restore, or clean changes yourself.
5. Continue until the tool says the run is complete, the maximum is reached, or no useful hypotheses remain. Finish with a concise summary of accepted commits, rejected experiments, and total gain.
6. Do not ask for confirmation between iterations.`, cfg.Objective, cfg.Benchmark, cfg.Direction, strings.Join(cfg.EditablePaths, ", "), cfg.MaxIterations)
}

func (a *app) stop() ext.Response {
	a.mu.Lock()
	wasActive := a.state.Active
	a.state.Active = false
	a.state.StoppedAt = time.Now().UTC()
	err := a.saveStateLocked()
	a.mu.Unlock()
	if err != nil {
		return ext.Errorf("autoresearch stop: %v", err)
	}
	a.renderPanel()
	if !wasActive {
		return ext.Display("Autoresearch is already stopped.")
	}
	return ext.Display("Autoresearch stopped. Existing accepted commits were kept.")
}

func (a *app) experimentTool(raw json.RawMessage) ext.ToolResult {
	var input experimentArgs
	if err := json.Unmarshal(raw, &input); err != nil {
		return ext.TextErrorResult("invalid arguments: " + err.Error())
	}
	input.Hypothesis = strings.TrimSpace(input.Hypothesis)
	input.Summary = strings.TrimSpace(input.Summary)
	if input.Hypothesis == "" || input.Summary == "" {
		return ext.TextErrorResult("hypothesis and summary are required")
	}
	result, err := a.runExperiment(input)
	if err != nil {
		a.ext.Notify("error", err.Error())
		return ext.TextErrorResult(err.Error())
	}
	a.renderPanel()
	return ext.TextResult(result)
}

func (a *app) setPanelOpen(open bool) {
	a.mu.Lock()
	a.panelOpen = open
	a.mu.Unlock()
}

func (a *app) panelKey(key, text string) {
	if key != "rune" {
		return
	}
	switch strings.ToLower(text) {
	case "r":
		a.renderPanel()
	case "s":
		_ = a.stop()
	}
}
