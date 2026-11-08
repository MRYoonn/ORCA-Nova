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
| Windows build/package/checksum | Pending release workflow evidence below |
| Stable release | Blocked by gates above |

## User data and scientific evidence

No private chemistry datasets are uploaded. ORCA binaries are not installed or redistributed. The public paper DOI 10.1002/anie.9565676 and data DOI 10.19061/iochem-bd-6-619 were retried; blocked retrieval cannot prove the requested source relationship or expected numerical values. Synthetic tests remain explicitly synthetic. Reaction profiles support 1:1 composition comparisons, not arbitrary weighted multimolecular thermochemistry. Missing XYZ composition must block automatic profiles.

## Candidate delivery

Tag v1.6.0-rc.1 will trigger the repository's existing authorized Windows release workflow. RC syntax is explicitly supported and forces --prerelease --latest=false so the stable update channel excludes it. Release assets retain ORCA_Nova_Portable_Windows.zip and SHA256SUMS.txt; Korean notes explain incomplete gates and separate-folder review.

## Resume instructions

Current HEAD: run `git rev-parse HEAD` on release/v1.6.0-rc.1. Test with Go 1.23: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build ./...`. Read the current Release tag/Actions status before retrying publication; never overwrite an existing version. After network access to api.github.com and the verified data hosts is available, fetch actual Golden fixtures under their public license and complete PR reviews in order. Licensed Windows ORCA and an isolated existing v1.5.1 installation are required for the unrun execution/update gates.
