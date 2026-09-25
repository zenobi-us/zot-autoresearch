# Autoresearch TUI design proposal

## Goal

Make the state of an autonomous optimization loop legible in under two seconds:

1. Is it still running?
2. Is the best result actually better than baseline?
3. How many candidates were accepted, rejected, or failed?
4. Which commit contains the current best result?
5. What has already been tried?

## Proposed panel

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

The implementation emits plain panel lines and lets zot own borders, clipping, colours, Unicode width, and terminal layout.

## Information hierarchy

### Header

The title carries only the run state: `running` or `stopped`. The user should not have to infer liveness from the latest row.

### Summary block

- **Objective** anchors the work and makes a stale/wrong run obvious.
- **Runs** gives progress and outcome totals in one scan.
- **Baseline** preserves the original comparison point.
- **Best** is visually adjacent to total gain and the accepted commit that realizes it.

Gain is always normalized so positive means better, regardless of whether the configured direction is `minimize` or `maximize`.

### Ledger

Rows use stable semantics:

| Mark | Status | Meaning |
| --- | --- | --- |
| `◆` | baseline | Original clean-tree measurement. |
| `✓` | accepted | Mechanically improved on the previous best and was committed. |
| `×` | rejected | Valid benchmark that did not improve enough; configured paths were restored. |
| `!` | crash/invalid | Command failed, timed out, or emitted no valid score; configured paths were restored. |

The panel shows the 12 most recent entries and reports how many earlier rows were omitted. This keeps the dialog useful on short terminals without pretending the panel is a full-screen dashboard.

### Footer

Only actions valid in the focused modal are shown:

- `r`: redraw from persisted in-memory state;
- `s`: stop the run but preserve accepted commits;
- `esc`: close the panel without stopping research.

## zot constraints

The current extension API provides a single focused modal panel, not a permanent sidebar or status widget. Therefore:

- the panel cannot remain visible while the normal editor has focus;
- only one extension panel can be active;
- panel content is not written to the chat transcript;
- state must be persisted by the extension;
- terminal dimensions and resize events are not currently exposed through the public Go SDK.

The design deliberately avoids interactions that need a cursor, horizontal scrolling, multiple panes, or resize-aware column calculation.

## Future design extensions

If zot gains richer panel APIs, the next useful additions are:

1. a selected-row detail view containing full hypothesis, summary, benchmark tail, and rejection reason;
2. a compact always-visible status line with current iteration and best gain;
3. a gain sparkline and confidence/noise band;
4. segment filtering for resumed or reconfigured runs;
5. an explicit pause/resume control distinct from stop.
