# Training UI simplification

Daily training now centers on the fixed list and current exercise. Ability
assessment and weekly practice volume live in Training Assessment. Source rows,
template tiers, allocation evidence, score predictions and recording diagnostics
remain available through details or help rather than competing with execution.

The overview combines actual plan completion and workbench guidance in one card.
Sensitivity and scenario distribution charts are available under History's more
analysis menu. Free-practice phase estimates are no longer a default overview.

The catalog defaults to training type, personal fit, feedback and scheduling
permission. Chinese type names are searchable; played/unplayed filters use the
shared history index. Local file and requirement diagnostics remain in advanced
details. Comparable scores and personal difficulty feedback no longer require a
verified local SCE. Feedback uses a keyboard-accessible dialog with optional
metadata editing, visible save errors and focus restoration.

## Benchmark training preview

The former independent REC scoring algorithm has been removed. The training
column reads the same `TrainingGuidance` provider as the workbench and overview.
Current/Next badges identify preview membership; hover explains role and remaining
runs. Repeated preview entries are combined, finished entries are omitted, and
stale/error guidance is not presented as a current arrangement.

Guidance is a bounded upcoming preview, not the full plan. An empty cell means
only that the scenario is absent from that preview. It does not mean excluded,
low priority or absent from the complete list. Viewing this column never changes
the generated plan or official benchmark ranks.

Group strength retains normalized rank progress and no longer averages raw
scores across different scenarios. Rank distribution separates a missing score
from recorded below-first-rank performance. Display controls remain in settings;
existing persisted preferences are respected.

## History and settings

History names its filter explicitly as a training-plan filter, displays an active
filter chip with a clear action, and switches between lists and details when the
available content width is below 1100px. Environment diagnostics and raw stats
are secondary to score, accuracy, duration and TTK. Persisted inspector values
remain compatible, including the environment view.

Automatic discovery preferences can be saved independently of a generated plan.
The new bridge changes only the global discovery preference. Recording buffer,
session grouping and app maintenance are secondary controls. Unavailable sync
switches are replaced by one local-storage message. History loading controls do
not delete records or change the assessment window. Welcome emphasizes practical
setup steps and keeps recording choices and reading resources optional.

New text is provided in Chinese and English. Other supported locales retain their
existing translations and use the English catalog for new labels.

## Validation

- `npm run test:catalog`: 31 passing tests, including Chinese/user catalog filters,
  shared benchmark arrangements and combined overview progress.
- `npm run build`: TypeScript and production Vite build passed.
- `go test -tags bindings ./...`: passed.
- `go run ./cmd/training-contracts -check`: passed.
- `go test -race ./internal/training ./internal/practice ./internal/runs`: passed.
- Research Python test discovery: 14 passing tests.
- Chromium with simulated desktop bridge: long played/unplayed names, missing
  SCE, fixed paused list, feedback save failure/retry, Escape and focus return,
  discovery preference persistence, history filter clearing, shared REC badges,
  unavailable sync, empty history and first-launch welcome passed.
- Thirty page/layout checks across five main pages, with the sidebar expanded
  and collapsed: 1280×720 at 100%, 1366×768 at 125%, and 1920×1080 at 150% app
  scale. Catalog and feedback dialog overflow checks passed.

The browser checks use fixture data and app scale, not actual Windows display
scaling or game integration. Windows CI validates the desktop executable; live
KovaaK's reminders, recording and playlist installation still require a Windows
game session.
