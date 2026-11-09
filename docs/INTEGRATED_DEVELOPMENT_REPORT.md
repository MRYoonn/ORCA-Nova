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

## Final evidence and commits

- `76a2f0b`: visualization integration, regression tests and initial report.
- `d309a33`, `052754d`: Windows test diagnostics, preserving actual failures rather than guessing.
- `83fdd89`: rooted ZIP path rejection fix for Windows.
- Final documentation commit records these outcomes without altering application code.

PR creation attempted with base `feat/batch-job-queue`; `api.github.com/graphql` returned Forbidden. No PR was created. Creation URL: https://github.com/MRYoonn/ORCA-Nova/compare/feat/batch-job-queue...feat/orca-nova-integrated-visualization?expand=1 . This depends on PR #12 until it merges. No merge is claimed.

Windows source commit `83fdd89` SUCCESS: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42497246326 (1m29s). Go tests, vet, Windows backend/icon, WebView2 native host and Portable ZIP packaging completed. Download the `ORCA-Nova-Windows-Build` artifact from that run; it contains `ORCA_Nova_Portable_Windows.zip` and `SHA256SUMS.txt`. Failed runs 42496832164 / 42496996881 / 42497110853 are retained and diagnosed in BUILD_FAILURE_ANALYSIS.md; their tiny failure artifacts are not portable builds.

Browser validation passed: existing Batch UI fixtures, actual Go server + four XYZ / concurrency2 / failed-job retry / isolated TXT-JSON-CSV reports, visualization fixtures and actual CSV/SVG/PNG downloads (nonempty PNG signature), all inline JavaScript syntax. Local Linux full test and race suites passed; final ZIP path fix additionally passed focused updater race tests and vet, followed by the successful Windows full suite.

Real ORCA scientific execution: NOT RUN. Real Windows GUI launch/MPI/automatic updater installation and restart: NOT RUN. No stable Release or environment publication.
