# ORCA Nova v1.10.0-rc.1

Windows Beta for the D9 HAT / NEB-TS workflow. Use the **Beta** update channel to receive trial releases.

- Review the transferring H, donor and acceptor before running D9. Inspect differences in the inferred bonds and optionally accept a unique Product H swap while retaining the original geometry.
- IRC reads the checked NEB-TS geometry after SCF/geometry convergence and a verified TS Gibbs/frequency result. It runs in both directions and compares final endpoint connectivity when both endpoint XYZs are available.
- Run all four stages, restart at NEB-TS, run IRC only, or resume. Verified outputs can be reused in a fresh folder. All stages support individual INP edits and shared settings.
- The Windows workflow ZIP contains BAT launchers and a standalone runner with the same checks as Nova. It requires an installed ORCA and its MPI runtime.
- Executable simple-input lines normalize legacy M06-2X to M062X while preserving valid hyphenated keywords. DEFGRID3 is the automatic M06-2X grid for general calculations. Literature settings are retained.
- Reports include stage Gibbs energies, barriers, the TS imaginary frequency, H distances, IRC connectivity and available spin diagnostics.

The release checks include mock ORCA workflows and existing real-output parser fixtures. Actual D9 execution with licensed ORCA 6.0.1 or 6.1.x, TS Hessian reuse and ORCA 7 are pending. [D9 instructions and scope](HAT_NEB_WORKFLOW.md)
