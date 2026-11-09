# Integrated development failure log

## External access (2026-11-08)

- `gh pr view 12 --json ...` and PR API calls: proxy/API 403 for `https://api.github.com/graphql`. Git HTTPS fetch/push works. Use the authorized Git path and provide a PR creation URL. Do not extract proxy credentials or request secret values in chat.
- Crossref metadata request: proxy CONNECT 403 for `api.crossref.org`. No Glorius SI-derived preset has been registered without verified title/DOI/page/settings. Existing general presets remain general presets.
- Required follow-up: allow the required domains in environment settings, then retry the specific read/API operation with existing authentication. Publisher/SI host requirements must come from the actual source, not guesses.

## Validation scope

Parser and calculation tests use explicitly synthetic fixtures and Mock ORCA. There is no installed licensed ORCA or Windows desktop on this Linux machine. GitHub Windows Actions validates Go tests, WebView2 host compilation and ZIP packaging; it does not establish scientific correctness or validate real MPI/update restart behavior.

New compile/test failures and fixes are recorded below as they occur. Existing historical failure analysis is retained in the repository root.

## Windows run 42496832164

Code commit 76a2f0b: Windows workflow failed after 53 seconds (exit 1). The public page did not expose test logs; authenticated API requests remain blocked by 403. No cause is claimed from the exit code alone. Added CI capture of verbose Go test output and a bounded public error annotation to obtain the actual failing test without bypassing access controls. Local Linux 73 test/subtest cases and race suite passed. Follow-up run and fix recorded below.

Run 42496996881 reproduced exit 1. Public diagnostics proved the new descendant cancellation and retry transaction tests passed on Windows. GitHub truncated the broad verbose annotation before the actual failing line, so the diagnostic annotation now selects only FAIL/panic/source-location lines. Full runner output remains captured. No assertions were disabled or success inferred.

## Confirmed Windows defect and fix

Run 42497110853 exposed `update_integration_test.go:128: unsafe payload accepted /absolute.dll`. Windows `filepath.IsAbs` does not reject drive-relative rooted paths the same way as Linux. Fixed archive validation to reject a leading slash or backslash in the original ZIP entry before platform path normalization. This preserves the safety assertion rather than skipping Windows coverage. Descendant cleanup, queue isolation, recovery and reports passed in the same Windows run. Local focused updater tests and follow-up Windows CI verify the correction below.

## Successful revalidation

Source fix commit 83fdd89 passed focused local updater race tests and vet. Windows run https://github.com/MRYoonn/ORCA-Nova/actions/runs/42497246326 completed SUCCESS in 1m29s, including full Go tests/vet, backend/icon, WebView2 host and complete portable packaging. Failed runs remain historical evidence; they are not reclassified as passing. Actual ORCA/MPI and real updater restart remain outside this validation.

## v1.6.0 RC release review

Fresh GitHub API and both requested DOI/PMC retrieval attempts returned 403; actual Golden numerical validation and reviewed PR merge operations remain blocked. Windows ORCA/MPI and real updater replacement/relaunch are unrun. Scientific review fixed ignored solvent/input blocks, missing XYZ composition admission, and TDDFT inputs being admitted to UKS/ground-state comparisons. Version review fixed RC→same-version stable upgrade detection. Focused race regressions pass; full results are in RELEASE_REGRESSION.md and RELEASE_VALIDATION_REPORT.md. Stable release is deliberately blocked; the explicitly authorized fallback is a prerelease with Korean limitations and --latest=false.

## RC publication outcome (2026-11-09 KST)

Source 9df25b5 passed full local Go test/race/vet/build. Windows Release run 42536452100 succeeded and published v1.6.0-rc.1 as Pre-release. Published ZIP was independently downloaded, CRC checked and matched against SHA256SUMS.txt (abbf9c8d3b4a9179a9d4d787a4e452b86db5979de5839b4124ff5b0712b18bd9). Stable latest remains v1.5.1. Stabilization PR creation remains blocked by GraphQL Forbidden; no PR merges or actual Windows/ORCA scientific verification are claimed. Required remaining gates are enumerated in RELEASE_VALIDATION_REPORT.md.

## 2026-11-09 — real ORCA Golden extension (new feature branch only)

- Test integration commit `7b8bc0be` failed: `real_orca_golden_test.go:52:1: expected declaration, found '}'`. Actual Windows and Linux job logs confirmed an extra closing brace. Commit `16ba806d` removed it; real ORCA water Single Point and Opt+Freq Golden test passed on run `42572683652`. Pre-existing RC source and tag were unchanged.
- Initial scan-coordinate implementation (`b5683ca7`, `5001b91b`) broke `visualization.go` with unexpected duplicated package text. A JavaScript replacement string unintentionally expanded its special replacement token when writing the Go file. The source was reconstructed from the last passing SHA, the patch was reapplied without global-text expansion and the valid parser was restored in `9b9e077e`. Windows validation passed in run `42573195134`.
- Additional Golden workflow first revision (`9f5d66dc`) contained malformed YAML due to the same replacement-string expansion issue. The workflow was reconstructed from the last valid version and regenerated in `23ca4d63`; Golden run `42573294339` passed.
- Genuine ORCA 6.1.1 calculations used transiently, not redistributed: public tutorial H₂O (SP, Opt+Freq), ORCA Result Analyzer test-suite ethane relaxed PES, benzene TD-DFT and triplet O₂. Source refs are pinned and Git blob identifiers checked by the CI downloader. Test cases assert independently read numerical values (see `real_orca_golden_test.go`, `real_orca_more_golden_test.go`).
- **Still not run:** local ORCA executable/MPI process tree, actual Windows GUI click-through, v1.5.1 automatic-update install/restart and comprehensive UI DLL rollback. Real output parsing is now partially Golden-tested; this does not satisfy runtime integration gates.
