# Batch per-molecule presets and readable Result Summary

## Before

The Result Summary displayed the complete analysis as JSON, making it hard to distinguish successful calculations, absolute energies, and unavailable measurements. The Batch table only allowed applying the current global preset to all molecules or editing generated INP manually.

## Changed

- Each Batch molecule gets an independent calculation preset dropdown grouped by the existing built-in calculation families, including A4 and W2. Choosing one regenerates **only that molecule's** requested files/stages, leaving the global selected calculation and other Batch entries unchanged.
- The global "현재 프리셋을 모두 적용" control retains its deliberate overwrite behavior, with a single confirmation if any custom INP is present. It explicitly sets every molecule's preset ID.
- Imported INP remains a distinct "Imported INP · 직접 편집" selection until the user explicitly chooses a built-in preset. The confirmation protects manually edited INP content.
- Previously submitted Queue jobs, originals and disk results are never mutated by editing Batch registrations. Existing batch Preflight, resource scheduling and browser regression remain enabled.
- The Result Summary now displays a human-readable status and provenance header, convergence indicators, a four-energy grid, optional <S²>, TD-DFT Vertical Root table, and Mulliken (or separately named available scheme) Atomic Spin Population sorted by absolute magnitude while retaining signed values and ORCA's **zero-based** atom indices. Warnings and full original JSON analysis are accessible in collapsible details. A plain-text summary copy action supports research notes.
- Distinguishes Electronic Energy, Gibbs Free Energy, Enthalpy and ZPE without converting absolute values into misleading relative energies. Missing numbers display **Not calculated**. It does not automatically infer C5/C6 identity, assign EnT feasibility, deduce reaction energy, or equate vertical excitation with optimized triplet energy.

## Validation and limitations

The new `tests/batch-presets-summary-ui.cjs` browser regression checks per-molecule isolation, stage generation, custom INP cancel/confirm, imported INP preservation, summary rendering, provenance, unavailable fields, HTML escaping, and regression against accidentally showing old values after loading another source. Existing Chromium Batch and XYZ 3D tests still run. Windows Go/WebView2/Portable CI checks independent packaging. Actual licensed ORCA/MPI and Windows user-machine click-through remain separate release criteria.

The summary is based exclusively on fields already parsed from an OUT file. It is not a scientific validation of the selected theoretical method or a comparison of independently optimized states.
