# zot-autoresearch plan

`zot-autoresearch` adapts the benchmark-driven loop from Karpathy's autoresearch for zot's extension protocol.

## First release

- [x] One `/autoresearch` command with a discrete subcommand router.
- [x] Project-local JSON, TOML, or YAML configuration with objective, benchmark, metric, direction, scope, timeout, and iteration limit.
- [x] An LLM-callable experiment tool that establishes a baseline, benchmarks candidates, mechanically compares scores, commits improvements, and restores rejected in-scope changes.
- [x] Durable per-project state in the extension data directory.
- [x] A live modal zot panel showing iterations, gains, accepted commits, rejections, and failures.
- [x] Guardrails for clean startup, explicit editable paths, bounded benchmark runtime, finite metric parsing, and out-of-scope change detection.

## Follow-up candidates

- [ ] Run candidates in temporary Git worktrees instead of the user's primary worktree.
- [ ] Add optional correctness checks that hard-block acceptance.
- [ ] Add asynchronous benchmark jobs for runs longer than zot's 60-second tool timeout.
- [ ] Add structured `METRIC name=value` support and secondary metrics.
- [ ] Add JSONL event history, segments, run logs, confidence/noise estimates, and export.
- [ ] Bundle an autoresearch setup skill and deterministic compaction context.
- [ ] Add bounded auto-resume when zot exposes a public extension submit API.
