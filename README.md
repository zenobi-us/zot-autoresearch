# zot-autoresearch

[![CI](https://github.com/zenobi-us/zot-autoresearch/actions/workflows/ci.yml/badge.svg)](https://github.com/zenobi-us/zot-autoresearch/actions/workflows/ci.yml)

Benchmark-driven autonomous research loops for [zot](https://github.com/patriceckhart/zot), inspired by [karpathy/autoresearch](https://github.com/karpathy/autoresearch), [pi-autoresearch](https://github.com/davebcn87/pi-autoresearch), and [ozeron/autoresearch](https://github.com/ozeron/autoresearch).

The extension gives the active zot agent a mechanically scored experiment tool. Each candidate is benchmarked against the current best: improvements are committed, regressions and failed runs are restored, and the live panel shows the experiment ledger.

## What it does

- Registers one namespaced command, `/autoresearch`, with a discrete subcommand router.
- Uses a project-local `.zot/autoresearch.json`, `.toml`, `.yaml`, or `.yml` contract for the objective, benchmark, score parser, direction, editable paths, timeout, and iteration limit.
- Establishes a baseline before candidates can be evaluated.
- Derives accept/reject decisions from the captured score rather than trusting the model.
- Commits accepted in-scope changes as `perf(autoresearch): …`.
- Restores rejected, crashed, or unparsable candidates only within configured paths.
- Refuses dirty starts and stops without cleanup when out-of-scope changes appear.
- Persists per-project run state in zot's extension data directory.
- Renders a live modal zot panel with iterations, improvements, gains, accepted commits, rejections, and failures.

## Requirements

- zot with extension tools and panels (tested against the SDK from zot `v0.3.69`)
- Go 1.25 for development or use through the source manifest
- Git
- A benchmark that finishes within 55 seconds and prints one parseable numeric score

A normal zot extension tool has a 60-second deadline. Longer benchmarks need the planned asynchronous job API.

## Install and run

```sh
git clone https://github.com/zenobi-us/zot-autoresearch.git
cd zot-autoresearch
go test ./...
go build -o zot-autoresearch .
```

For development, load the repository directly:

```sh
zot --ext /path/to/zot-autoresearch
```

The checked-in manifest uses `go run .`. To install a built binary, change `extension.json` to use `"exec": "./zot-autoresearch"`, then run:

```sh
zot ext install /path/to/zot-autoresearch
```

## Quick start

1. In the target Git repository, run:

   ```text
   /autoresearch init reduce parser allocations without changing behaviour toml
   ```

   The final argument selects `json`, `toml`, `yaml`, or `yml`; everything between `init` and the format becomes the configured objective.

2. Edit and commit the generated `.zot/autoresearch.toml`. The benchmark must emit text matched by `score_pattern`; capture group 1 must be the numeric score.
3. Ensure the repository is clean.
4. Start the loop:

   ```text
   /autoresearch start reduce parser allocations without changing behaviour
   ```

5. Open the dashboard when desired:

   ```text
   /autoresearch watch
   ```

The start command submits an operating prompt to the current zot agent. The agent establishes the baseline, edits code, and calls `autoresearch_experiment` after every candidate.

## Slash command design

All actions are routed through one command to keep zot's slash namespace small:

| Command | Action |
| --- | --- |
| `/autoresearch help` | Show the command reference. |
| `/autoresearch init <goal> <format>` | Create a configuration with the supplied objective. Formats: `json`, `toml`, `yaml`, or `yml`. Calling it without arguments retains the default objective and JSON format. |
| `/autoresearch start [objective]` | Validate Git/configuration and submit the autonomous loop prompt. An objective argument overrides the configured objective for this run. |
| `/autoresearch watch` | Open the live panel. |
| `/autoresearch status` | Show compact totals and the current best score. |
| `/autoresearch history` | Open the same persisted ledger panel after a run. |
| `/autoresearch stop` | Stop accepting experiments while preserving accepted commits. |

Unknown subcommands fail with a focused error and point back to `/autoresearch help`.

## Configuration

`/autoresearch init <goal> <format>` creates the selected configuration and sets `objective` to the supplied goal. For backward compatibility, `/autoresearch init` with no arguments creates `.zot/autoresearch.json` with the default objective:

```json
{
  "objective": "Improve the benchmark score without changing its semantics.",
  "benchmark": "go test ./... -run '^$' -bench . -count 1",
  "score_pattern": "(?m)^score:\\s*([-+]?[0-9]*\\.?[0-9]+)",
  "direction": "minimize",
  "unit": "score",
  "editable_paths": ["."],
  "max_iterations": 20,
  "timeout_seconds": 55,
  "min_delta": 0
}
```

The loader also accepts the same keys from `.zot/autoresearch.toml`, `.zot/autoresearch.yaml`, or `.zot/autoresearch.yml`. Keep exactly one autoresearch configuration file; startup fails rather than silently choosing one when multiple formats are present.

The default benchmark is only a starting point: standard Go benchmark output does not contain a `score:` line. Wrap your real benchmark so it prints one, for example:

```text
score: 4331
```

| Option | Meaning |
| --- | --- |
| `objective` | Durable research goal and constraints. |
| `benchmark` | Shell command run from the project root. Treat it as executable code. |
| `score_pattern` | Go regular expression; capture group 1 is parsed as a finite `float64`. |
| `direction` | `minimize` or `maximize`. |
| `unit` | Display suffix such as `ms`, `µs`, `tokens/s`, or `score`. |
| `editable_paths` | Project-relative Git pathspecs the loop may stage or restore. |
| `max_iterations` | Candidate limit; the baseline is not counted. |
| `timeout_seconds` | Per-run timeout, from 1 through 55 seconds. |
| `min_delta` | Required absolute improvement over the current best. |

## TUI proposal

`/autoresearch watch` opens an extension-owned modal panel. It remains live while focused; zot currently does not expose a permanently docked sidebar API.

```text
┌ Autoresearch · running ─────────────────────────────────────────────────────┐
│ Objective  reduce parser allocations without changing behaviour            │
│ Runs       6/20     ✓ 3 accepted     × 2 rejected     ! 1 failed           │
│ Baseline   7,574 µs                                                        │
│ Best       4,331 µs     +42.82% gain     091534f                           │
│                                                                            │
│ #   result     score          gain       commit    hypothesis              │
│ ────────────────────────────────────────────────────────────────────────── │
│ 0   ◆ baseline   7,574 µs       —          —         establish baseline    │
│ 1   ✓ accepted   6,102 µs       +19.44%    3799d4c   cache token lookup    │
│ 2   × rejected   6,441 µs       +14.96%    —         preallocate cursor    │
│ 3   ! crash      —              —          —         parallel parse stage  │
│ 4   ✓ accepted   5,018 µs       +33.74%    0b07487   avoid scan fallback   │
│ 5   × rejected   5,203 µs       +31.31%    —         compact cursor state  │
│ 6   ✓ accepted   4,331 µs       +42.82%    091534f   skip duplicate lookup │
├────────────────────────────────────────────────────────────────────────────┤
│ r refresh · s stop · esc close                                             │
└────────────────────────────────────────────────────────────────────────────┘
```

See [`docs/TUI_DESIGN.md`](docs/TUI_DESIGN.md) for the visual hierarchy and interaction rationale.

## Safety model

This extension executes the configured benchmark and Git commands with the user's permissions. It is an automation guardrail, not a sandbox.

Before starting, it requires:

- a Git repository with an initial commit;
- valid configuration;
- a clean configured scope (the baseline tool then requires the entire tree to remain clean).

During a run it:

- limits the benchmark to 55 seconds or less and terminates its process tree on timeout;
- requires the benchmark itself to leave the Git working tree unchanged;
- rejects missing, invalid, NaN, and infinite scores;
- stages and restores only `editable_paths`;
- stops and leaves the tree untouched if out-of-scope changes are detected;
- leaves an improved candidate intact if committing fails, rather than destroying work.

For maximum isolation, temporary worktrees and optional correctness checks are tracked in [`PLAN.md`](PLAN.md).

## Development

```sh
gofmt -w *.go
go test ./...
go vet ./...
go build ./...
```

Protocol stdout belongs to zot. Runtime diagnostics go to stderr and can be read with:

```sh
zot ext logs zot-autoresearch
```

## Status

Experimental first release. The score/commit/revert loop and panel are implemented; long-running asynchronous jobs, correctness gates, worktree isolation, structured multi-metrics, and auto-resume are planned.
