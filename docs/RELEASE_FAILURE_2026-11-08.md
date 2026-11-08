# Release build failure analysis — 2026-11-08

## Failure
Release workflow run 42483507042 failed immediately with **zero jobs**, while ordinary Windows Portable Build passed.

## Root cause
An unsafe JavaScript string-replacement operation interpreted the `$'` token inside a PowerShell regular-expression string as a replacement suffix, duplicating part of the workflow YAML. Consequently, `.github/workflows/release.yml` was syntactically malformed.

## Fix
- Reconstruct the file from the last known-good Release workflow.
- Use replacement callbacks to keep literal `$'` and `$` tokens in PowerShell code.
- Verify there is exactly one validation, Go-test, build, and publish step.
- Modify release request manifest to retrigger a new reviewed merge into main.
- Keep an automatic build-failure report artifact for steps that execute.

## Verification
PR Windows Portable Build must pass; then merge to main and verify Release workflow creates nonempty jobs, publishes v1.5.1 ZIP and SHA256SUMS, and `/releases/latest` returns v1.5.1.

Live Windows updater installation still needs a device-level test.
