# ORCA Nova v1.6.0-rc.1 release validation

## Decision

Prepare a development/validation Pre-release, not stable v1.6.0. Latest stable was freshly verified as v1.5.1; no v1.6.x tag existed at start. Base: cb6bc66c43becccdafddabf2f79cb4d49b7740f3 on feat/cross-job-profile-golden-validation-20261109. Release branch: release/v1.6.0-rc.1.

## PR review and integration

Fresh remote refs: #12 e54d82e, #15 08b5852, #16 aefb789, #17 cb6bc66. Public PR pages show Draft status; main remains 7752bf8. The release branch contains the stacked source through #17. No PR was merged or base changed: GitHub API returns Forbidden, and direct main push is not a substitute for the requested reviewed PR merges. Resume #12 → #15 → #16 → #17 after API access and required review. Review focus includes earlier queue/update regression work plus the current scientific and release regressions in GOLDEN_RELEASE_REVIEW.md and RELEASE_REGRESSION.md.

## Release gates

| Gate | Evidence/status |
|---|---|
| PRs merged | Not completed; Draft PRs and blocked API |
| Real ORCA Golden results | Not completed; requested DOI/PMC sources return proxy CONNECT 403; no real OUT fixture claimed |
| Windows ORCA/MPI computation/cancellation | Not run; Linux machine has no ORCA or Windows runtime |
| Actual v1.5.1 Windows updater replace/restart | Not run; compilation/mock tests do not satisfy this gate |
| Full updater rollback | Known limitation: backend EXE rollback does not transactionally restore all UI DLLs |
| Local software regression | Results recorded below and in RELEASE_REGRESSION.md |
| Windows build/package/checksum | PASS: run 42536452100, published assets downloaded and SHA-256 verified |
| Stable release | Blocked by gates above |

## User data and scientific evidence

No private chemistry datasets are uploaded. ORCA binaries are not installed or redistributed. The public paper DOI 10.1002/anie.9565676 and data DOI 10.19061/iochem-bd-6-619 were retried; blocked retrieval cannot prove the requested source relationship or expected numerical values. Synthetic tests remain explicitly synthetic. Reaction profiles support 1:1 composition comparisons, not arbitrary weighted multimolecular thermochemistry. Missing XYZ composition must block automatic profiles.

## Candidate delivery

Tag v1.6.0-rc.1 triggered the repository's existing authorized Windows release workflow. RC syntax is explicitly supported and forces --prerelease --latest=false so the stable update channel excludes it. Release assets retain ORCA_Nova_Portable_Windows.zip and SHA256SUMS.txt; Korean notes explain incomplete gates and separate-folder review.

## Resume instructions

Current HEAD: run `git rev-parse HEAD` on release/v1.6.0-rc.1. Test with Go 1.23: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build ./...`. Read the current Release tag/Actions status before retrying publication; never overwrite an existing version. After network access to api.github.com and the verified data hosts is available, fetch actual Golden fixtures under their public license and complete PR reviews in order. Licensed Windows ORCA and an isolated existing v1.5.1 installation are required for the unrun execution/update gates.

## Published RC — final evidence (2026-11-09 KST)

- Published source commit: `9df25b510507484573d6b31541209b2228229344` (tag `v1.6.0-rc.1`). Subsequent report-only commits do not replace the tag or binaries.
- Release: https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.0-rc.1 — public page verified Pre-release label.
- Windows Release workflow SUCCESS: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42536452100 . This includes Windows Go tests, vet, backend/icon, WebView2 host, packaging and publication.
- ZIP: https://github.com/MRYoonn/ORCA-Nova/releases/download/v1.6.0-rc.1/ORCA_Nova_Portable_Windows.zip
- Checksums: https://github.com/MRYoonn/ORCA-Nova/releases/download/v1.6.0-rc.1/SHA256SUMS.txt
- SHA-256: `abbf9c8d3b4a9179a9d4d787a4e452b86db5979de5839b4124ff5b0712b18bd9`.
- Independently downloaded both published assets. Digest exactly matches SHA256SUMS.txt; ZIP CRC test passes; 3,654,102 bytes / 10 entries. Root ORCA_Nova.exe and ORCA_Nova_UI.exe present. No file named orca.exe is bundled.
- Stable latest URL remains https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.5.1 . RC did not replace the stable channel.
- Local full Go tests, race, vet and build passed after scientific changes; existing Batch/visualization/browser export and actual HTTP Mock Batch suites passed. See RELEASE_REGRESSION.md. These do not constitute actual ORCA/Windows GUI verification.
- PR merges: none. A stabilization PR creation attempt against PR #17's branch also returned GraphQL Forbidden. Creation URL: https://github.com/MRYoonn/ORCA-Nova/compare/feat/cross-job-profile-golden-validation-20261109...release/v1.6.0-rc.1?expand=1 . Do not claim a PR number or merged state.
- Actual ORCA Golden values, actual Windows ORCA/MPI calculations/cancellation, real v1.5.1 updater replacement/restart and complete UI-DLL rollback remain incomplete. Stable v1.6.0 remains blocked. This deliverable is an explicitly labeled development/validation Pre-release.
