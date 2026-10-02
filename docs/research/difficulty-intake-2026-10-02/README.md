# SCE intake and dimension-aware session planning

## Intake

The user's `Scenarios.zip` contains 58 SCE files with unique internal names.
All 30 first-family acquisition targets are present. The original archive and
raw SCE files remain outside the repository; the inventory records their SHA256
hashes and explicit root fields. This is an uploaded content snapshot, not proof
of the current Workshop version or independently verified gameplay.

53 files have at least one declared mechanism or explicit scoring-constraint
tag; five have no supported tags. Counts overlap because tags are multi-label:

| Tag | Files |
| --- | ---: |
| Short transfer | 4 |
| Wide transfer | 6 |
| Micro adjustment | 10 |
| Precision | 8 |
| Time pressure | 9 |
| Reflex window | 4 |
| Increasing pace | 3 |
| Phased targets | 6 |
| Tracking | 6 |
| Projectile | 3 |
| Accuracy constraint | 23 |
| Kill-replenished magazine constraint | 22 |

Tags come from matched phrases in the embedded author description or explicit
player weapon fields. No tags are inferred from scenario titles. Descriptions
are declarations and can be stale. In particular, the VT ww5t Intermediate
description says 40 ammo returned on hit, while the weapon field records
`AmmoReloadedOnKill=37`; retain the explicit value without claiming which behavior
the engine currently produces. A magazine/refill tag does not quantify penalty.

The parser now follows movement and weapon ability references. Pressure targets
can have zero ordinary movement speed and still have an Approach ability; zero
speed must not establish a stationary classification. Three ww5t files have
case-variant profile references. Candidate dependencies are resolved while
retaining diagnostics; this does not verify engine case semantics. Melee ability
references, map geometry, auxiliary bot spawns, phase triggers and actual ability
activation remain unsupported. All 58 retain `calibrationEligible=false`.

## Planning behavior

The embedded snapshot enriches matching catalog entries, including after
application restart. It does not import or enable arbitrary extra scenarios.
Previously unclassified imported EvoClick/StrawberryClick entries can use their
explicit author-declared static category. Manual classifications and durations
remain authoritative. The declared time limit provides a catalog estimate;
three or more comparable recent runs still override it with observed duration.

The session reserves 10% for loading/rest and gives half the remaining budget to
preparation and half to practice/exploration/measurement. Each phase keeps whole
runs. An underfilled preparation phase does not transfer its budget to practice.
If candidates are insufficient, warnings disclose the unused budget.

Within existing broad skill priorities, the selector favors under-covered demand
tags based on completed time in the previous seven days and the proposed plan.
A multi-label scenario divides its exposure time among its demand tags; it does
not count as several complete runs. Consecutive scenarios with similar demand
tags are downweighted, alongside existing family and recent-repeat controls.

Pressure/pace scenes are limited to one run per block. Where an eligible tagged
non-pressure alternative fits the budget, they are excluded from preparation and
from positions immediately following another pressure scene. If no such
alternative exists, pressure remains selectable. Missing tags do not prove low
load. A future explicitly verified warmup-only role excludes that exercise from
practice and measurement; all current uploaded role assignments remain unknown.

The reserved benchmark is completed once at the end, without a pass threshold.
KovaaK's exported list is fixed; no live rewriting is introduced. Reasons expose
the tags and their unverified snapshot status. No new manual selection is needed.

Playlist recommended player tiers are no longer inherited as individual scene
difficulty. Existing non-manual `playlist` difficulty labels are reset to unknown
on load/merge; manual and benchmark labels are preserved.

## Boundaries and next calibration work

These selection weights are bounded product heuristics, not a fitted difficulty
formula, fatigue diagnosis, or fine-grained weakness diagnosis. Raw scores across
scenes and raw world-space radii across maps are never equated. Angular size and
transition angle remain null. Several Voltaic wide-wall variants lack an explicit
wide-transfer declaration in their file descriptions and remain untagged for that
specific demand rather than relying on their titles.

Next: validate spawn/map coordinate transforms and active ability/phase rules;
derive angular-demand distributions; then calibrate within mechanism families
against benchmark strata. Cross-family calibration requires performance bridges
and held-out families/versions, not playlist-level labels.

## Reproduction

From repository root, with the original archive available:

```sh
python docs/research/difficulty-acquisition-2026-10-02/tools/intake_sce.py /path/to/Scenarios.zip --snapshot internal/training/data/sce-mechanics-2026-10-02.json --inventory docs/research/difficulty-intake-2026-10-02/inventory.json
python -m unittest discover -s docs/research/difficulty-acquisition-2026-10-02/tools -p 'test_*.py'
go test ./internal/training
```

Go tests cover phase budgets and fixed measurement across 100 random seeds,
pressure repetitions/adjacency, dimension-history effects, mixed tags, unknown
geometry, warmup-only roles, and playlist-tier migration. The Windows workflow
builds the frontend, tests the backend, and compiles the desktop application.
