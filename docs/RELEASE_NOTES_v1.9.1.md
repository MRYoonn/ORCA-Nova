# ORCA Nova v1.9.1 — Stable hotfix

## Fixes
- Emit the ORCA 6.x `M062X` keyword instead of the literature label `M06-2X` across general DFT, Batch, Glorius presets, D9 HAT workflow and Triplet Geometry Transfer.
- Preserve the familiar UI label and detect obsolete spelling during preflight.
- Order Reaction Mechanisms D8 before D9.
- Update input examples and add regression tests.

## Validation
PR #59 Windows Portable Build and XYZ Manager / Batch Browser QA passed. The versioned stable package is rebuilt and smoke-tested by the release workflow. A real D9 HAT calculation remains unverified. Previously generated INP files are not silently overwritten; regenerate them from the corrected preset.
