# Release regression review

Review base: `cb6bc66c43becccdafddabf2f79cb4d49b7740f3` (latest cross-job branch fetched by parent). Reviewed cumulative changes for PR #12 (`e54d82e`), #15 (`08b5852`), #16 (`aefb789`), #17 (`cb6bc66`) using their actual Git diffs, with focused inspection of retry isolation, update trust/staging and current profile regression coverage.

## Executed checks

- `go test -count=1 ./...`: PASS on Linux.
- `go test -race -count=1 ./...`: PASS on Linux (24.274s).
- `go vet ./...`: PASS.
- `go build -o /tmp/orca-release-regression .`: PASS.
- `node tests/scientific-ui-syntax.cjs`: PASS, four embedded scripts plus profile controls and DOI guards.
- `node tests/batch-ui.cjs`: PASS, synthetic API fixtures with four XYZ and preserved existing UI behavior.
- `node tests/visualization-ui.cjs`: PASS, synthetic plot data and CSV/SVG/PNG export including PNG signature.
- Actual HTTP Mock Batch result recorded below after completion.

Full Go/race execution includes queue cancellation/recovery/resource/isolation tests and scientific/cross-job fixtures. These fixtures are not real ORCA Golden data. Browser tests used Chromium, not Windows WebView2.

## Confirmed issue and fix

The updater stripped prerelease suffixes for comparison, causing stable `1.6.0` to be considered equal to current `1.6.0-rc.1`; RC users would not be offered the final stable version. `newerVersion` now considers a final release newer than a same-core prerelease while preserving downgrade protection. Added `TestStableReleaseReplacesSameVersionRC`, including RC-to-stable, stable equality, prerelease candidate and downgrade cases. Focused updater/version race suite passes. Parent release commit will identify the fix; this review agent does not commit independently.

## Required gates not satisfied by this machine

Host is Linux. `orca` and `wine` are absent from PATH. No ORCA binary was installed, license bypass attempted, or third-party executable bundled. Actual ORCA/MPI calculations and actual Windows WebView2/portable update-relaunch tests were NOT RUN.

Current updater installation script backs up/restores the backend EXE only; rollback is not transactional across every native UI DLL. This is an unresolved stable-release blocker until corrected and tested on Windows. Compile success and mocked ZIP staging do not validate actual process shutdown, replacement or relaunch. Settings preservation tests cover staging; they are not an actual v1.5.1-to-release upgrade.

Actual HTTP + Chromium Mock Batch integration: PASS. Four XYZ W2 workflows, two simultaneous jobs, isolated folders, one failed job retried, TXT/JSON/CSV/summary artifacts and no browser JavaScript errors. Evidence directory: `/tmp/orca-e2e-Tczqu8`. Backend used `/tmp/orca-release-regression`; this is Mock ORCA, not a real quantum-chemistry computation.
