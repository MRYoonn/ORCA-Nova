# D9 HAT / NEB-TS + Gibbs + IRC

D9 prepares a two-endpoint intramolecular H transfer. The default neutral doublet and the selected geometries are hypotheses to validate with the transition mode, IRC and spin diagnostics.

## Desktop workflow

1. Import the Reactant as the primary XYZ. Register the Product as `product.xyz` in the additional XYZ picker.
2. Select **1,3-HAT · NEB-TS + Gibbs + IRC**. Enter the transferring H, donor and acceptor indices (ORCA 0-based).
3. Use **원자 대응 검사**. The review shows inferred bond additions/removals and both structures aligned on the donor, acceptor and a common off-axis heavy atom when one is available. Covalent distances are used to infer the graph. Confirm bonding manually for unusual structures. Supported elements are those in the app's Triplet Transfer covalent-radius table.
4. If exactly one H exchange restores the intended single-bond loss and gain, Nova can preview and save a corrected Product. Applying it requires confirmation. If more than one exchange fits, manually inspect the correspondence. The originally imported file stays available as `product_original.xyz` in the new run when a correction was accepted.
5. Select shared charge, multiplicity, Functional, Basis Set, Dispersion, Integration Grid, Solvation and temperature. Generated M06-2X uses `M062X` and defaults to `DEFGRID3`. The Glorius preset calculations retain their existing literature-level settings. Individual D9 stage edits are retained across stage changes. **공통 설정으로 전체 INP 재생성** reapplies the common settings to all stages. Preflight checks edited inputs and compares shared settings.
6. Run all four stages. The optimized endpoint graphs must still match each starting structure. For NEB-TS, the runner expects the final `<stem>_NEB-TS_converged.xyz` seen in the reported ORCA 6.0.1 output. IRC uses it only after normal termination, SCF and geometry convergence, Gibbs energy and one imaginary mode. A generic `<stem>.xyz`, NEB-CI image or intermediate geometry cannot satisfy this gate.

## Restart and standalone ZIP

Select **Run Step 3–4**, **Run IRC Only**, or **Resume from Failed Stage**, and supply a previous run immediately under the configured work root. Restart compares the approved atom map, original endpoint XYZs, scientific INP and digests of verified outputs. Compatible completed stages are copied to a fresh folder. Reused INPs are copied verbatim, including their previous PAL/MaxCore. GBW, intermediate NEB files and unverified Hessians are excluded. A failed prior stage's files stay in its original run. Runs from older versions without verification records must be started again.

**Save Workflow ZIP** in the Windows Portable app exports the two XYZs, the current stage INPs, the no-install Windows runner, a README and four BAT launchers. Extract the full set, set `ORCA_EXE` to the absolute executable path (or put ORCA on PATH), and launch the appropriate BAT. For a partial run set `ORCA_RESUME_DIR` to a previous run under the extracted folder's `HAT_results`. The runner and desktop use the same Preflight and validation. Read `HAT_RUN.log` and the new run's OUT, ERR and analysis files.

## Interpreting the results

The report and JSON retain Reactant/Product/TS Gibbs energies in Eh and calculate reaction ΔG and forward/reverse ΔG‡ in kcal/mol at a common detected output temperature. It also presents the TS imaginary mode, indexed H distances and any available ⟨S²⟩/Mulliken spin population. Intermediate HEI energies describe the NEB search. Barriers are provisional until the imaginary mode and IRC connectivity have been inspected. When forward and backward final endpoint XYZs are present, the report compares their indexed bond graphs to the specified endpoints.

## Validation scope

The Functional registry targets the app's exposed ORCA 6.0/6.1 presets. Browser tests audit all generated stages and Go tests exercise the workflow and both successful and failed checkpoints using a mock ORCA process. The installed ORCA 6.0.1/6.1.x and MPI runtime, version-specific IRC output variants, TS Hessian reuse and ORCA 7 remain to be verified. The runner leaves Hessian handling to ORCA's IRC defaults.

[ORCA 6.1 NEB-TS tutorial](https://www.faccts.de/docs/orca/6.1/tutorials/react/nebts.html) · [ORCA 6.1 IRC manual](https://www.faccts.de/docs/orca/6.1/manual/contents/structurereactivity/irc.html)
