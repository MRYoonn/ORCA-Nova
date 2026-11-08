# ORCA Nova integrated development

## Base and PR dependency

Fresh Git remote reads at task start showed main `7752bf88209891f99e10023e27a92028f612f5a8`, Batch/PR #12 head `e54d82ef7776bfa5c9f1cdcf7f876320b67c022f`. Public PR #12 was Open. This branch is based on that exact Batch head, not a speculative main merge. Existing Batch functionality is retained.

Integrated branch: `feat/orca-nova-integrated-visualization`. The intended PR base is `feat/batch-job-queue` while #12 is open; after #12 merges, retarget to main and resolve against its actual new head. No direct main push or stable Release is authorized by validation here.

## Scope

- Offline SVG/Canvas visualization in the existing results area; explicit units and source trace.
- PES Scan discovery/parsing and raw/relative energy charts, with unknown convergence kept unknown.
- Existing structured OUT analysis reused for state, atomic spin, TD-DFT and frequency displays.
- Queue/process/update regressions without duplicating #12's scheduler.
- Existing dark theme, presets and Korean help retained.

## Scientific limitations

No generated demonstration curve substitutes for missing calculation data. TD-DFT triplet roots are not UKS optimized triplets. Comparisons must retain Method/Geometry and energy type. Adiabatic states/reaction mechanisms cannot be confirmed from an arbitrary pair of outputs. Spin populations use zero-based ORCA indices; users map chemical labels such as C5/C6 themselves.

PES two-column energy tables alone do not prove convergence or bond-distance units; explicit input/OUT context must establish those. Unsupported output formats remain unavailable. Real ORCA 6.x Golden samples were not available; parser fixtures are synthetic and labeled accordingly.

## Files and verification

Final commit, test and Windows build evidence will be filled after integration. See BUILD_FAILURE_ANALYSIS.md for access failures and unresolved prerequisites. No environment publication or stable release is performed.

## Implemented files and checks

- `visualization.go`, `visualization_test.go`: bounded OUT/DAT discovery, explicit scan coordinates, stage convergence, Optimization/gradient/SCF/IR series, raw source and safe XYZ retrieval. Synthetic parser/endpoint tests.
- `assets/desktop.html`: four results tabs, offline SVG plots, tables, raw OUT, hover/click provenance, PES reference and units, actual state/Spin/TD-DFT/Frequency views, CSV/SVG/PNG export, dropdown readability. No layout replacement.
- `queue.go`, `queue_test.go`: retry failure transaction preservation, nonregular artifact exclusion, real descendant heartbeat cancellation tests. Existing PR12 Batch tests retained.
- `update.go`, `update_integration_test.go`: mock network/download/digest/archive coverage, root executable validation fix, bounded downloads and settings staging preservation. Full UI DLL rollback remains incomplete; actual Windows install/restart not executed.
- `main.go`: authenticated visualization route alongside existing APIs.
- Reusable browser tests: `tests/visualization-ui.cjs`, `tests/batch-ui.cjs`, `tests/batch-http.cjs`; use NODE_PATH pointing to Playwright and optional CHROMIUM_PATH.

Local validation: `go test -count=1 ./...` passed 73 test/subtest cases; `go test -race -count=1 ./...`, `go vet ./...`, `go build ./...` passed. All inline JavaScript syntax checked with Node. Browser evidence and Windows artifact status follow below after final integration.

## Explicitly unfinished requirements

Automatic Adiabatic S1/T1 and Singlet–Triplet Gap derivation across separately assigned states, verified reaction role assignment/barrier profiles, per-scan-step atomic population extraction, and cross-Job S0/T1 scan overlay with a scientifically validated common reference are not complete. Current visualizations expose measured series with provenance and warnings, not invented state assignments or conclusions. Optimization/SCF tables and generic labeled scan formats are supported; unrecognized ORCA variants need real Golden fixtures.

Glorius SI templates remain blocked on verified public sources; no literature preset is fabricated. Windows native UI execution, real ORCA/MPI cleanup and scientific validation, and full automatic updater installation/restart/rollback remain unrun. Neither a stable Release nor environment publication is performed.
