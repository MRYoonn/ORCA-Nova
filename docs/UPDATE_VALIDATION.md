# Automatic updater validation

## Changes

The existing GitHub Releases update channel, TLS verification, SHA-256 verification, HTTPS redirect allowlist and Windows installer are retained. Download HTTP transport can now be injected for deterministic offline tests; normal operation still uses the default verified transport. Archive staging and SHA-256 validation were extracted into helpers exercised on Linux and Windows.

A discovered payload-validation defect was fixed: a nested `subdir/ORCA_Nova.exe` previously satisfied the required-backend check even though installation launches the root executable. Only a root `ORCA_Nova.exe` now satisfies that requirement.

## Tests performed

`go test -race -run '^TestUpdater' -count=1 ./...` passed with six top-level tests and subcases:

- Valid newer stable release offered; draft, prerelease and incomplete assets rejected.
- Offline refresh clears stale installable release metadata and resets Checking.
- Download byte limits, 403 responses, trusted source URLs, HTTPS-only redirects and allowed redirect hosts.
- Correct SHA-256 accepted; changed bytes rejected.
- Portable archive extraction with native host and WebView2 runtime DLL subdirectories.
- Traversal/absolute/drive paths, scripts, unexpected executables, duplicate paths and missing/rootless backend rejected.
- Existing `config.json` in the staging fixture remains unchanged by a permitted portable payload.

Fixtures contain synthetic bytes. No live release was installed or published. The tests do not assert that preservation in a staging fixture proves preservation by a live Windows installation.

## Remaining device validation

The PowerShell replacement/restart script, real Windows file locking, runtime authentication, actual update download, settings preservation across a live replacement and rollback after a partially copied native-host payload still require Windows device validation. Current rollback restores the backend EXE; it is not a transactional rollback of every UI/runtime DLL. These limitations must remain documented and this development branch must not be presented as a scientifically validated stable release.
