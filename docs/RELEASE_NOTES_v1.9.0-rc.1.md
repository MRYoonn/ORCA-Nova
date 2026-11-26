# ORCA Nova v1.9.0-rc.1 (Beta)

## New: D9 — HAT / NEB-TS + Gibbs + IRC
- Two atom-mapped XYZ endpoints: reactant and product.xyz.
- Independent endpoint Opt/Freq, NEB-TS/Freq, and IRC workflow; no manual TS guess required.
- Neutral doublet UKS M06-2X/def2-TZVP, CPCM(DMF), TightSCF defaults; calculation settings can be changed.
- Preflight verifies matching atom element order, charge, multiplicity, method, temperature and stage dependencies.
- Stationary-point checks and automatic reaction, forward and reverse Gibbs energy differences in kcal/mol.

## Validation and limits
- PR #58 Windows Portable Build, Browser QA and ORCA 6.1.1 Golden Validation passed. These existing suites do not constitute a real D9 HAT NEB-TS calculation.
- A real D9 calculation on ORCA 6.x, the H-transfer imaginary mode and IRC endpoint connectivity remain to be verified. Atom identity among atoms of the same element requires manual review.
- D9 is a desktop workflow; independent molecule Batch mode is not supported. ZIP export does not provide automatic scientific validation or Gibbs reporting.
- Existing stable v1.8.0 remains the stable update target. This release is a prerelease.

[Usage guide](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/HAT_NEB_WORKFLOW.md)
