# ORCA Nova

A Korean-friendly, lightweight **unofficial** Windows frontend for ORCA quantum chemistry.

ORCA Nova does not redistribute ORCA and is not affiliated with its developers.

## Features
- Import XYZ geometries from Avogadro; generate and edit ORCA `.inp` files.
- DFT, TD-DFT, S0/S1/T1, geometry optimization, frequency, spin-density presets and multi-step workflows.
- Korean help tooltips for calculations, functionals, basis sets, solvents and other options.
- Select a locally installed `orca.exe`, run calculations, view `.out` logs, and inspect failure reports.
- **계산 전 검사 (Preflight):** Check XYZ atom counts/coordinates, Charge/Multiplicity electron parity, source file references and upstream Optimization dependencies without running ORCA.
- Native WebView2 host with an ORCA Nova icon, Windows portable distribution.
- Check GitHub Releases for updates; install a new version only after user approval and SHA-256 check.

> **Validation:** Input templates, spin multiplicities, TD-DFT roots, solvent-model names, convergence and final electronic state must be reviewed for scientific correctness before interpreting output. Actual ORCA 6.x calculations have not yet been verified end-to-end on Windows.

## Run (Windows)
1. Download `ORCA_Nova_Portable_Windows.zip` from a verified Release or an Actions artifact.
2. Unzip the **whole** archive into a writable folder. Run `ORCA_Nova.exe`.
3. Use ORCA Nova's Settings to point to the full path of an independently installed `orca.exe`.
4. Import an XYZ file and select a preset. Review the generated INP before running.
5. For multi-step workflows, make sure all required source XYZ files are created by preceding steps.

Windows 10/11, .NET Framework 4.8, and Microsoft Edge WebView2 Runtime are required for the native UI. If unavailable, Edge app mode may be used; its taskbar icon cannot be guaranteed.

The default job directory is `Documents/ORCA Nova Jobs`; preferences live in `%APPDATA%/ORCA Nova/config.json`. Legacy ORCA Studio preferences will be read on first launch if no new configuration exists.

## Building
- **Backend:** Go 1.23; no third-party Go packages.
- **Native Windows frontend:** .NET Framework 4.8 WinForms and Microsoft.Web.WebView2.
- **CI:** `.github/workflows/windows-build.yml` builds/test artifacts on every push to `main` and manually (`workflow_dispatch`).
- **Release:** `.github/workflows/release.yml` packages a versioned ZIP and checksums on `v*` tag push or **Actions > Release Windows Portable > Run workflow** on `main`.
- **First pilot release:** Choose `release_tag: v1.4.0` and leave `prerelease: true` while Windows/ORCA validation is incomplete. Prereleases do not appear in the stable auto-updater. For a verified stable Release, select `prerelease: false` for a *new* version tag.
- **Update source:** `https://github.com/MRYoonn/ORCA-Nova/releases/latest`.

To build the Go backend locally, run `go test ./...` then `go build .`. The Windows native host and icon packaging are built by GitHub Actions on Windows.

## Security / privacy
- The app binds its local HTTP service to `127.0.0.1` and requires a per-launch session token.
- No ORCA calculation inputs or outputs are sent to GitHub. Only the updater requests public Release metadata and assets.
- Do not commit private calculation datasets, unpublished structures, credentials, or large ORCA outputs.
- SHA-256 only checks agreement with release-provided checksums; it does **not** replace code-signing or a trusted publisher verification.
- Updates are disabled during an active job and require explicit approval.

## Status
Source migration from ORCA Studio to ORCA Nova. CI build and native taskbar icon require Windows-run verification before labeling a Release stable.

## Next development milestones
- P0: Real ORCA 6.x calculation smoke tests on Windows, state/Root validation, reliable rollback of portable updates.
- P1: Resume interrupted multi-step workflows, job queue, failure analysis per stage.
- P1: Parse OUT energies, optimized geometry, imaginary frequencies, spin population and spin contamination into a Korean results summary.
- P2: Compare Vertical/Adiabatic S1/T1 gaps with clearly labeled methods and units; support multiple XYZ batch runs.
