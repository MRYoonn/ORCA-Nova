# ORCA Nova v1.9.0 — Stable

User-approved stable application release, promoting the v1.9.0-rc.1 code without changes to its calculation implementation.

## D9 — HAT / NEB-TS + Gibbs + IRC
- Load two atom-mapped XYZ endpoints, with the product named product.xyz.
- Independent reactant/product Opt/Freq, NEB-TS/Freq and IRC; no manual TS guess.
- Neutral doublet UKS M06-2X/def2-TZVP, CPCM(DMF), TightSCF defaults; adjustable settings.
- Preflight checks for matching atom element order, electronic state, method, temperature and stage dependencies.
- Stationary-point checks and automatic reaction, forward and reverse Gibbs energy differences in kcal/mol.

## Validation
The release candidate passed the backend tests and vet, Windows Portable Build, Browser QA, existing ORCA 6.1.1 Golden Validation, packaging/checksum generation and packaged Windows EXE/API smoke test. The versioned stable binary will be rebuilt and smoke-tested before publication.

## Scientific and operational limitations
Actual D9 HAT NEB-TS execution on ORCA 6.x is still unverified. Inspect the imaginary vibrational mode, IRC endpoint connectivity, spin state and atom mapping manually; element order alone does not establish identity among atoms of the same element. D9 supports the desktop workflow, not independent-molecule Batch mode. ZIP export does not supply automatic scientific validation or Gibbs reporting. Stable denotes the application distribution channel, not validation of the proposed HAT mechanism.

[Usage guide](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/HAT_NEB_WORKFLOW.md)
