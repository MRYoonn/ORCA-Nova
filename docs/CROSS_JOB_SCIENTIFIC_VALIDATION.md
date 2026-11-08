# Cross-Job Scientific Profile Validation — 2026-11-09

## Scope
Stacked on PR #16 (which depends on #15 and #12). This is a developer-only branch; **not a stable release**.

### Implemented
- S₀, T₁, Reactant, TS and Product roles can each select a different **existing Batch Job ID** and OUT filename in the Scientific Profile UI. Existing single-Job behavior is unchanged.
- The backend resolves each source from the application's registered queue, not from user-specified filesystem paths.
- Identical filenames in different jobs are supported; assigning one identical OUT within the same Job to two roles is rejected.
- If local starting XYZ geometries are present and valid, atom-by-atom elemental composition **counts** are compared; nonidentical composition is rejected even if total electron counts agree (e.g. C+O versus N+N).
- This automatic formula check is necessary, not sufficient. The user still confirms conformer/molecular identity, correct charge/multiplicity and Spin Surface, state identity, reaction stoichiometry, TS frequencies and IRC, common calculation method and consistent energy definitions.
- Reaction Profile remains a 1:1 structure comparison, not arbitrary weighted multi-component reaction energies. No missing energies or fictitious curves are synthesized.

### Tests
- Cross-job job-ID resolution, same OUT basename in two independently registered jobs.
- Rejection of repeated role assignment to same physical OUT, unknown/host filesystem paths, traversal strings and unexpected role keys.
- Rejection of method mismatch across jobs and mismatched element composition with equal total atomic number.
- Go regressions, JS syntax, Windows binary/WebView2/portable packaging via GitHub Actions.

### Data source discovery — Glorius Hartmann 2026
Hartmann et al., *Energy-Transfer Catalysis Enables the Birch-Type Reduction of 2-Pyridones*, DOI https://doi.org/10.1002/anie.9565676.

Open-access PMC article links original supplemental PDF and data ZIP:
- https://pmc.ncbi.nlm.nih.gov/articles/instance/13427111/bin/ANIE-65-e9565676-s001.pdf
- https://pmc.ncbi.nlm.nih.gov/articles/instance/13427111/bin/ANIE-65-e9565676-s002.zip
- https://doi.org/10.19061/iochem-bd-6-619 (authors' actual ORCA input/output archives)

These resources are documented only as **located references**. Download from the current automation environment was blocked by network/browser restrictions, so no extracted real-ORCA golden samples or claim of fully verified SI calculations has been made. Do not distribute the user's unpublished chemistry to public GitHub.

### Remaining blockers before stable release
1. Retrieve legally accessible public ORCA OUT/INP fixtures and verify parser against actual ORCA 6.x outputs, including root parsing, PES steps, Spin, frequency and thermochemistry.
2. Test real ORCA/MPI Windows job cancellation and child-process cleanup, including queue recovery after abnormal termination.
3. Test Windows native GUI opening, actual local updater install / restart / rollback, and checksum handling end-to-end.
4. Review exact SI settings and ORCA keyword equivalence for method-specific presets, especially thermochemical corrections and state identity.
5. Merge stacked PRs in order after review; never infer a successful native calculation from Windows CI compilation.

## No-force conclusion
Windows CI success confirms software build and synthetic regression tests, **not** chemical accuracy or ORCA/MPI integration.
