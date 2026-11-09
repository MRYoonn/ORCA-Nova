# ORCA Nova v1.6.0-rc.2 — pre-release gate report

## Release decision
Pre-release candidate only. Real ORCA 6.1.1 Golden output parsing has been validated for five publicly available calculations. Windows isolated PowerShell updater success/rollback regression and WebView2 compilation have passed. Actual installed licensed ORCA/MPI execution/cancellation and the full end-user v1.5.1 updater/relaunch are **not** validated, so stable v1.6.0 is explicitly blocked.

## Verified source and tests
- All integration PRs #12, #15, #16, #17, #18, #19 merged in order to main. GitHub confirmed respective successful merges; last main merge at #19 was `cd5c71e`.
- PR #20 (all-file transactional portable rollback) merged to main at `11f123ca4dee48b48ffa4dc9d03ecd0af541e64f`.
- Real ORCA Golden CI SUCCESS: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42573626339
- Windows updater transactional success/forced-rollback CI SUCCESS: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42574801764
- Final merged `main` Windows CI SUCCESS: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42575117822
- The public files used for real Golden tests were downloaded transiently and were never included in commits or portable distribution. See `docs/REAL_ORCA_GOLDEN_VALIDATION.md`.
- PowerShell 5.1 UTF-8 BOM failure and repair are recorded in `docs/BUILD_FAILURE_ANALYSIS.md`.

## Release control
- `docs/RELEASE_NOTES_v1.6.0-rc.2.md` contains the precise RC.2 changes and known limitations.
- The approved main release-request manifest triggers Windows Release; RC naming must force `--prerelease --latest=false`.
- A stable v1.6.0 **must not** be published until the remaining real Windows/ORCA/MPI and installed-update checks have been evidenced.
- The published v1.6.0-rc.1 tag/assets and v1.5.1 stable release must remain unchanged.

## Still unverified
1. Licensed Windows ORCA/MPI actual end-to-end calculations and process tree stop.
2. Actual v1.5.1 Portable installation upgrade/restart/rollback and preservation of user's settings/jobs.
3. Broad real ORCA chemical calculations beyond the five pinned ORCA 6.1.1 examples, plus complete Glorius SI method reproduction.

## Post-publication record
After GitHub Release Actions completes, verify actual public assets `ORCA_Nova_Portable_Windows.zip`, `SHA256SUMS.txt`, published version, prerelease flag, Windows CI and stable latest release. Append evidence as a separate follow-up commit if needed; do not silently rewrite a published tag.
