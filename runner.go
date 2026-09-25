package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const maxStoredOutput = 4096

func (a *app) runExperiment(input experimentArgs) (string, error) {
	a.runMu.Lock()
	defer a.runMu.Unlock()

	a.mu.Lock()
	if !a.state.Active {
		a.mu.Unlock()
		return "", errors.New("autoresearch is not active; run /autoresearch start")
	}
	cfg := a.state.Config
	baselineMissing := a.state.Baseline == nil
	candidateCount := countCandidates(a.state.Iterations)
	if !baselineMissing && candidateCount >= a.state.MaxIterations {
		a.state.Active = false
		a.state.StoppedAt = time.Now().UTC()
		_ = a.saveStateLocked()
		a.mu.Unlock()
		return "", errors.New("maximum iterations reached; the run is complete")
	}
	a.mu.Unlock()
	if err := validateConfig(cfg); err != nil {
		return "", fmt.Errorf("saved run configuration is invalid; restart the run: %w", err)
	}
	if err := validateEditablePaths(a.cwd, cfg.EditablePaths); err != nil {
		a.stopForSafety()
		return "", err
	}

	allDirty, err := dirtyPaths(a.cwd, nil)
	if err != nil {
		return "", fmt.Errorf("inspect working tree: %w", err)
	}
	scopeDirty, err := dirtyPaths(a.cwd, scopedPaths(cfg.EditablePaths))
	if err != nil {
		return "", fmt.Errorf("inspect editable paths: %w", err)
	}
	if baselineMissing && allDirty != "" {
		return "", fmt.Errorf("baseline requires a clean working tree; commit or stash these changes:\n%s", allDirty)
	}
	if !baselineMissing {
		if allDirty == "" {
			return "", errors.New("candidate has no changes; make one focused edit before running an experiment")
		}
		if normalizeStatus(allDirty) != normalizeStatus(scopeDirty) {
			a.stopForSafety()
			return "", fmt.Errorf("changes outside editable_paths detected; run stopped without modifying the working tree:\n%s", allDirty)
		}
	}

	output, runErr := runShell(a.cwd, cfg.Benchmark, time.Duration(cfg.TimeoutSeconds)*time.Second)
	postAllDirty, statusErr := dirtyPaths(a.cwd, nil)
	if statusErr != nil {
		return "", fmt.Errorf("inspect working tree after benchmark: %w", statusErr)
	}
	postScopeDirty, statusErr := dirtyPaths(a.cwd, scopedPaths(cfg.EditablePaths))
	if statusErr != nil {
		return "", fmt.Errorf("inspect editable paths after benchmark: %w", statusErr)
	}
	if normalizeStatus(postAllDirty) != normalizeStatus(postScopeDirty) {
		a.stopForSafety()
		return "", fmt.Errorf("benchmark or candidate changed files outside editable_paths; run stopped without modifying the working tree:\n%s", postAllDirty)
	}
	if normalizeStatus(postAllDirty) != normalizeStatus(allDirty) {
		a.stopForSafety()
		return "", fmt.Errorf("benchmark modified the working tree; run stopped without cleanup. Benchmarks must be read-only:\n%s", postAllDirty)
	}
	if runErr != nil {
		if baselineMissing {
			return "", fmt.Errorf("baseline benchmark failed: %v\n%s", runErr, tail(output, maxStoredOutput))
		}
		return a.rejectFailed(input, cfg, "crash", output, runErr.Error())
	}
	score, err := parseScore(output, cfg.ScorePattern)
	if err != nil {
		if baselineMissing {
			return "", fmt.Errorf("baseline score: %v\n%s", err, tail(output, maxStoredOutput))
		}
		return a.rejectFailed(input, cfg, "invalid", output, err.Error())
	}

	if baselineMissing {
		a.mu.Lock()
		a.state.Baseline = floatPtr(score)
		a.state.Best = floatPtr(score)
		a.state.Iterations = append(a.state.Iterations, iteration{
			Number: 0, Status: "baseline", Score: score, Hypothesis: input.Hypothesis,
			Summary: input.Summary, Output: tail(output, maxStoredOutput), CreatedAt: time.Now().UTC(),
		})
		err := a.saveStateLocked()
		a.mu.Unlock()
		if err != nil {
			return "", fmt.Errorf("save baseline: %w", err)
		}
		return fmt.Sprintf("Baseline established: %s. Form a focused hypothesis, edit only configured paths, then call this tool again.", formatMetric(score, cfg.Unit)), nil
	}

	a.mu.Lock()
	best := *a.state.Best
	baseline := *a.state.Baseline
	number := countCandidates(a.state.Iterations) + 1
	a.mu.Unlock()
	gain := gainPercent(score, baseline, cfg.Direction)
	if improved(score, best, cfg.Direction, cfg.MinDelta) {
		commit, err := acceptChanges(a.cwd, scopedPaths(cfg.EditablePaths), input.Summary)
		if err != nil {
			a.stopForSafety()
			return "", fmt.Errorf("benchmark improved to %s but commit failed; run stopped and changes were left intact: %w", formatMetric(score, cfg.Unit), err)
		}
		a.mu.Lock()
		a.state.Best = floatPtr(score)
		a.state.BestCommit = commit
		a.state.Iterations = append(a.state.Iterations, iteration{
			Number: number, Status: "accepted", Score: score, Gain: gain, Hypothesis: input.Hypothesis,
			Summary: input.Summary, Commit: commit, Output: tail(output, maxStoredOutput), CreatedAt: time.Now().UTC(),
		})
		complete := number >= a.state.MaxIterations
		if complete {
			a.state.Active = false
			a.state.StoppedAt = time.Now().UTC()
		}
		err = a.saveStateLocked()
		a.mu.Unlock()
		if err != nil {
			return "", fmt.Errorf("save accepted result: %w", err)
		}
		message := fmt.Sprintf("ACCEPTED #%d %s (%+.2f%% from baseline), commit %s. %s", number, formatMetric(score, cfg.Unit), gain, commit, input.Summary)
		if complete {
			message += " Maximum iterations reached; summarize the run."
		}
		return message, nil
	}

	if err := restorePaths(a.cwd, scopedPaths(cfg.EditablePaths)); err != nil {
		a.stopForSafety()
		return "", fmt.Errorf("candidate regressed to %s and rollback failed; run stopped: %w", formatMetric(score, cfg.Unit), err)
	}
	a.mu.Lock()
	a.state.Iterations = append(a.state.Iterations, iteration{
		Number: number, Status: "rejected", Score: score, Gain: gain, Hypothesis: input.Hypothesis,
		Summary: input.Summary, Output: tail(output, maxStoredOutput), CreatedAt: time.Now().UTC(),
	})
	complete := number >= a.state.MaxIterations
	if complete {
		a.state.Active = false
		a.state.StoppedAt = time.Now().UTC()
	}
	err = a.saveStateLocked()
	a.mu.Unlock()
	if err != nil {
		return "", fmt.Errorf("save rejected result: %w", err)
	}
	message := fmt.Sprintf("REJECTED #%d %s (%+.2f%% from baseline); configured paths restored to best commit. Try a materially different hypothesis.", number, formatMetric(score, cfg.Unit), gain)
	if complete {
		message += " Maximum iterations reached; summarize the run."
	}
	return message, nil
}

func (a *app) rejectFailed(input experimentArgs, cfg config, status, output, reason string) (string, error) {
	if err := restorePaths(a.cwd, scopedPaths(cfg.EditablePaths)); err != nil {
		a.stopForSafety()
		return "", fmt.Errorf("experiment failed and rollback failed; run stopped: %w", err)
	}
	a.mu.Lock()
	number := countCandidates(a.state.Iterations) + 1
	a.state.Iterations = append(a.state.Iterations, iteration{
		Number: number, Status: status, Hypothesis: input.Hypothesis, Summary: input.Summary,
		Output: tail(output, maxStoredOutput), CreatedAt: time.Now().UTC(),
	})
	if number >= a.state.MaxIterations {
		a.state.Active = false
		a.state.StoppedAt = time.Now().UTC()
	}
	_ = a.saveStateLocked()
	a.mu.Unlock()
	return fmt.Sprintf("REJECTED #%d (%s: %s); configured paths restored. Diagnose the failure before the next hypothesis.\n%s", number, status, reason, tail(output, 1200)), nil
}

func (a *app) stopForSafety() {
	a.mu.Lock()
	a.state.Active = false
	a.state.StoppedAt = time.Now().UTC()
	_ = a.saveStateLocked()
	a.mu.Unlock()
}

func runShell(dir, command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	name, args := "sh", []string{"-c", command}
	if runtime.GOOS == "windows" {
		name, args = "cmd.exe", []string{"/d", "/s", "/c", command}
	}
	cmd := exec.CommandContext(ctx, name, args...)
	configureProcess(cmd)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output.String(), fmt.Errorf("benchmark timed out after %s", timeout)
	}
	return output.String(), err
}

func parseScore(output, pattern string) (float64, error) {
	match := regexp.MustCompile(pattern).FindStringSubmatch(output)
	if len(match) < 2 {
		return 0, errors.New("benchmark output did not match score_pattern")
	}
	score, err := strconv.ParseFloat(match[1], 64)
	if err != nil || math.IsNaN(score) || math.IsInf(score, 0) {
		return 0, fmt.Errorf("captured score %q is not a finite number", match[1])
	}
	return score, nil
}

func improved(score, best float64, direction string, minDelta float64) bool {
	if direction == "maximize" {
		return score > best+minDelta
	}
	return score < best-minDelta
}

func gainPercent(score, baseline float64, direction string) float64 {
	denominator := math.Abs(baseline)
	if denominator == 0 {
		return 0
	}
	if direction == "maximize" {
		return (score - baseline) / denominator * 100
	}
	return (baseline - score) / denominator * 100
}

func countCandidates(iterations []iteration) int {
	count := 0
	for _, item := range iterations {
		if item.Status != "baseline" {
			count++
		}
	}
	return count
}

func floatPtr(value float64) *float64 { return &value }

func tail(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return "…" + value[len(value)-limit:]
}

func normalizeStatus(value string) string {
	lines := strings.Fields(strings.TrimSpace(value))
	return strings.Join(lines, " ")
}
