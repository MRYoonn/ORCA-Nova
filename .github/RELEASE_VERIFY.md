# ORCA Nova automatic-update verification

Release workflow: merge a reviewed `.github/release-request.json` change into `main`. A push to main triggers Windows tests, portable build, checksums, and GitHub Release publication for the new tag.

Manual release: Actions > Release Windows Portable > Run workflow on main with a new version tag. Pre-releases do not appear in the stable updater.

## Acceptance checks on a Windows PC
1. Launch an older ORCA Nova portable build from a writable folder and press '업데이트 확인'.
2. Confirm the new tag is offered. Install; close/restart and confirm the version changed.
3. Confirm ORCA path and work directory preferences survive.
4. Attempt update during an active ORCA job: it must be blocked.
5. Test offline startup/check: failure should be nonfatal.
6. Compare ZIP SHA-256 with the Release checksum file.
7. Test a real ORCA calculation end-to-end. Windows build success alone is not scientific validation.

No unpublished molecular structures or job output should be uploaded to public Releases.
