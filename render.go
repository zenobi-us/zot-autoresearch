package main

import (
	"fmt"
	"strings"
)

func (a *app) panel() (string, []string, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return panelForState(a.state)
}

func panelForState(state runState) (string, []string, string) {
	accepted, rejected, failed := totals(state.Iterations)
	status := "stopped"
	if state.Active {
		status = "running"
	}
	title := fmt.Sprintf("Autoresearch · %s", status)
	lines := []string{
		fmt.Sprintf("  Objective  %s", compact(state.Objective, 76)),
		fmt.Sprintf("  Runs       %d/%d     ✓ %d accepted     × %d rejected     ! %d failed", countCandidates(state.Iterations), state.MaxIterations, accepted, rejected, failed),
	}
	if state.Baseline == nil {
		lines = append(lines, "  Baseline   waiting for first benchmark")
	} else {
		lines = append(lines, fmt.Sprintf("  Baseline   %s", formatMetric(*state.Baseline, state.Unit)))
		if state.Best != nil {
			gain := gainPercent(*state.Best, *state.Baseline, state.Direction)
			commit := state.BestCommit
			if commit == "" {
				commit = "baseline"
			}
			lines = append(lines, fmt.Sprintf("  Best       %s     %+.2f%% gain     %s", formatMetric(*state.Best, state.Unit), gain, commit))
		}
	}
	lines = append(lines, "", "  #   result     score          gain       commit    hypothesis", "  ─────────────────────────────────────────────────────────────────────────────")
	start := 0
	if len(state.Iterations) > 12 {
		start = len(state.Iterations) - 12
		lines = append(lines, fmt.Sprintf("  … %d earlier runs", start))
	}
	for _, item := range state.Iterations[start:] {
		icon := "·"
		switch item.Status {
		case "accepted":
			icon = "✓"
		case "rejected":
			icon = "×"
		case "crash", "invalid":
			icon = "!"
		case "baseline":
			icon = "◆"
		}
		commit := item.Commit
		if commit == "" {
			commit = "—"
		}
		score := "—"
		gain := "—"
		if item.Status != "crash" && item.Status != "invalid" {
			score = formatMetric(item.Score, state.Unit)
			if item.Status != "baseline" {
				gain = fmt.Sprintf("%+.2f%%", item.Gain)
			}
		}
		lines = append(lines, fmt.Sprintf("  %-3d %s %-9s %-14s %-10s %-9s %s", item.Number, icon, item.Status, score, gain, commit, compact(item.Hypothesis, 54)))
	}
	if len(state.Iterations) == 0 {
		lines = append(lines, "  No benchmark results yet.")
	}
	return title, lines, "r refresh · s stop · esc close"
}

func (a *app) renderPanel() {
	a.mu.Lock()
	open := a.panelOpen
	state := a.state
	a.mu.Unlock()
	if !open {
		return
	}
	title, lines, footer := panelForState(state)
	a.ext.RenderPanel(panelID, title, lines, footer)
}

func (a *app) statusText() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	accepted, rejected, failed := totals(a.state.Iterations)
	status := "stopped"
	if a.state.Active {
		status = "running"
	}
	best := "not measured"
	if a.state.Best != nil {
		best = formatMetric(*a.state.Best, a.state.Unit)
	}
	return fmt.Sprintf("Autoresearch %s — %d/%d iterations, %d accepted, %d rejected, %d failed, best %s.", status, countCandidates(a.state.Iterations), a.state.MaxIterations, accepted, rejected, failed, best)
}

func totals(iterations []iteration) (accepted, rejected, failed int) {
	for _, item := range iterations {
		switch item.Status {
		case "accepted":
			accepted++
		case "rejected":
			rejected++
		case "crash", "invalid":
			failed++
		}
	}
	return
}

func formatMetric(value float64, unit string) string {
	if unit == "" {
		return fmt.Sprintf("%.6g", value)
	}
	return fmt.Sprintf("%.6g %s", value, unit)
}

func compact(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit < 2 {
		return value[:limit]
	}
	return value[:limit-1] + "…"
}
