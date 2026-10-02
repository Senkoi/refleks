# Spawn geometry and mechanism audit

This continues the uploaded SCE intake. Run:

```sh
python docs/research/difficulty-acquisition-2026-10-02/tools/analyze_geometry.py /path/to/Scenarios.zip --targets docs/research/difficulty-acquisition-2026-10-02/data/first-family-targets.json --output docs/research/difficulty-geometry-2026-10-02/geometry-audit.json
python -m unittest discover -s docs/research/difficulty-acquisition-2026-10-02/tools -p 'test_*.py'
```

## Results

All 58 maps can be read structurally: 26 use KMC JSON and 32 use legacy Reflex
text. JSON objects retain explicit locations, rotations, team masks, profile
restrictions and weights. Legacy entity blocks retain explicit properties;
omitted team flags are unknown rather than guessed defaults. Spawn count is not
unique position count: overlapping entries and their multiplicity are retained.

12 scenarios permit bounded **conditional spawn-center calculations** under
explicit team/profile/shape assumptions. These are not validated gameplay angles.
Remaining scenarios are blocked by format semantics, rotations, profile shape,
slot alignment or missing required facts. Every authoritative angle remains null,
and every scenario remains ineligible for empirical difficulty calibration.

The first 30 acquisition targets produce 81 within-candidate-family pairs.
32 pairs have identical map payloads. Eight pairs differ only in bounding-box
size after excluding editorial names/descriptions and the explicit author label
fields `DifficultyTag` and `AimSubTypeTag`. These eight correlated pairs come from
four families, not eight independent experiments:

| Family | Configuration controls | Supported inference |
| --- | --- | --- |
| EvoClick | Entry / Int / unsuffixed: three pairs | Smaller main hitboxes increase precision demand under fixed runtime |
| StrawberryClick | Entry / Int / unsuffixed: three pairs | Same precision-demand ordering, with the mixed phase structure retained |
| Vibrating targets | Entry / unsuffixed: one pair | Smaller main hitboxes; vibration remains part of the shared configuration |
| Small flicks | Larger / slightly larger: one pair | Smaller main hitboxes under the same serialized mechanics |

This is a configuration control, not proof of equal runtime semantics or a
measured total-difficulty ordering. Older small-flick variants differ in save
versions, explicit/missing fields and reference lists; they cannot silently be
treated as a radius-only experiment. Missing fields are never replaced by zero.

## Important multi-variable differences

VT S5 small-wall scenarios use the same map but alter target radius, initial
target count (4 / 3 / 2), player spawn-block FOV (9 / 12 / 15), and target blocked
spawn radius (190 / 220 / 250). A radius-only difficulty function would discard
the altered transition and target-choice constraints.

VT S5 wide-wall Novice and Intermediate use 35 candidate target spawn entries;
Advanced uses 47. Their base-center maximum pair angles under the stated
coordinate model are approximately 90 / 90 / 100.37 degrees. The near-zero
minimum reflects overlapping/nearly overlapping authored entries, not proof of
valid zero-distance target transitions: collision and rejection rules intervene.

Revosect pressure scenarios share the map and four initial bot slots, but vary
target size and the Approach ability's velocity: 3500 / 4000 / 4444 raw config
units. The Int version's Approach condition includes target distance 2000 to
1000000 and FOV 1 degree; Depart uses distance 0 to 1000 and FOV 10 degrees.
Damage abilities also have self-health and distance gates. Travel distance divided
by Approach velocity alone therefore does **not** establish a reaction deadline.

The profile parser now follows weapon abilities into their referenced weapons,
and resolves explicitly supported melee ability references. Reachability still
does not prove that an ability fires; AI use gates, bot frequency, ability charges,
cooldowns, damage reactions and phase/map interactions must be considered.

## Conditional geometry model

For a spherical hitbox radius `r` and center distance `d`, angular diameter is
`2 asin(r / d)`. The angle between two normalized spawn-center directions is
`acos(dot(u, v))`. These analytic formulas are tested on known synthetic geometry.

Two coordinate-scale interpretations are preserved side by side: map coordinates
scaled with an unscaled profile radius, and map coordinates/radius sharing scale.
They can differ by several-fold. Output assumes spawn centers, no camera/offset
displacement, preserved axes, ignored object scale/rotation, and no collision or
spawn rejection. It reports center-angle extrema, never an empirical distribution
of played transitions. No such values enter the application's difficulty or
planning weights yet.

## Calibration gate and next work

Official documentation confirms that spawn offsets are relative to map spawn
points, that team/profile restrictions select spawns, and that blocked spawn
radius/FOV and AI use settings matter. It does not specify the numerical
serialization conventions needed to validate this parser's engine transforms.
Sources consulted on 2026-10-02:

- https://wiki.kovaaks.com/home/KovaaK%27s/ScenarioCreation/CharacterProfiles
- https://wiki.kovaaks.com/en/home/KovaaK%27sMapCreator/map-editing
- https://wiki.kovaaks.com/en/home/KovaaK%27s/ScenarioCreation/AbilityProfiles

Before fitting, validate map scaling, legacy team defaults, coordinate order,
spawn-offset frame, camera origin and spawn rejection against independent editor
or gameplay observations. Then use within-family benchmark strata as ordinal
constraints, preserve phase/pressure features, and hold out complete families and
versions. Three community tiers do not uniquely identify a multi-feature formula.
Cross-family difficulty needs performance bridges; no shared raw-score average
or leaderboard-top-100 population proxy is introduced.
