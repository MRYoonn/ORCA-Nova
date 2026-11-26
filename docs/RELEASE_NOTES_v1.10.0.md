# ORCA Nova v1.10.0 — Stable

User-approved stable application promotion of the tested v1.10.0-rc.1 source (6f01f7b13c3169b810b44537c2464121fe41e69e), merged through PR #60. No later feature-branch changes are included.

## D9 HAT workflow
- Specify transferring H, donor and acceptor; inspect bond changes and confirm a unique proposed Product H swap. Preserve the original geometry and recheck mapping after optimization.
- Pass only the verified converged NEB-TS geometry to bidirectional IRC; compare final endpoint connectivity when available.
- Run all stages, restart at NEB-TS, run IRC only, or resume. Reuse validated results in a fresh folder. Edit stage INPs and common settings.
- Export standalone Windows ZIP/BAT workflow with validation checks. Requires installed ORCA and MPI.
- Report Gibbs energies, barriers, imaginary frequency, H distances, endpoint connectivity and available spin diagnostics; omit unverified barriers.

## Input compatibility
Normalize M06-2X to executable M062X, including historical input aliases, while preserving legitimate hyphenated keywords. General M06-2X calculations default to DEFGRID3; literature settings remain unchanged.

## Validation and limits
The exact beta commit passed Windows Portable Build, Chromium QA and release packaging/EXE smoke tests. The stable package is independently rebuilt and smoke-tested before publication. D9 workflow testing used mock ORCA and existing real-output parser fixtures. Actual licensed ORCA 6.0.1/6.1.x D9 execution, version-dependent artifacts and Hessian reuse, and ORCA 7 remain unverified. Stable refers to the distribution channel, not scientific validation of a HAT mechanism.

[Usage guide](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/HAT_NEB_WORKFLOW.md)
