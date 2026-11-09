# Real ORCA 6.1.1 Golden Validation — independent public results

## Result

**Passed real output parser and visualization data tests** for five independently computed ORCA 6.1.1 outputs. Golden validation on GitHub Actions: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42573294339 (success). Windows package compilation is a separate check.

The tests use real `.out` files, downloaded transiently in CI and **never committed to this repository or included in the Portable ZIP**. Actual ORCA executable/MPI **was not run here**. This is a parser correctness check, not scientific reproduction of underlying calculations.

## Sources

1. Public teaching outputs (actual ORCA 6.1.1): https://github.com/dhwan0112/dhk.github.io/tree/0d7afec3fb8ca34b287e2413fccb4a7376899a75/guides/orca/examples  
   - `01-single-point/water.out` + input, Git object `32d1c6a0ad0bede114a6b55e546f7e6c023f5295`
   - `02-opt-freq/water.out` + input, Git object `21369a90c0a4a9683fce07888db5383491ad0c8a`
2. Independent ORCA Result Analyzer public parser fixtures (real ORCA 6.1.1): https://github.com/HiroYokoyama/moleditpy_orca_result_analyzer_plugin/tree/5a991e87847391f883e5668299424195dd1bef96/tests/sample_outputs  
   - `ethane-scan.out` (Git object `82ef1eab674d147eebab37cc7e4ba6e1ee375a14`)
   - `benzene-opt-vex.out` (Git object `87da13b827028b4d9683648d2600b8a01caef45f`)
   - `o2-opt_orb.out` (Git object `81d0e145e268889beedc5e8d2fb75c03b1b40139`)

All original outputs remain subject to their authors'/ORCA's licensing conditions. The workflow verifies immutable Git blob IDs, removes downloaded data even on test failure, and does not publish them as artifacts.

## Independently read reference values and assertions

| Source | Asserted actual result |
|---|---|
| H₂O Single Point | FINAL SINGLE POINT ENERGY = -76.321274411145 Eh; terminated and SCF converged |
| H₂O Opt+Freq | E = -76.426369457144 Eh; Final Gibbs = -76.42286885 Eh; Enthalpy = -76.40144400 Eh |
| H₂O Opt+Freq | ZPE = +0.02114567 Eh; T = 298.15 K; 0 imaginary modes |
| Ethane Relaxed PES | 5 steps with dihedral -60°, -30°, 0°, 30°, 60°; electronic energies matched within 1e-4 Eh |
| Benzene TD-DFT | Singlet roots 1–5 at 5.494, 6.469, 7.485, 7.485, 7.715 eV |
| Triplet O₂ | S² about 2.007028; 0 O and 1 O Mulliken spin populations about +1.0 each |

Real PES step coordinates come from ORCA's actual `RELAXED SURFACE SCAN STEP` banner. The graph labels `Bond` in Å and `Angle`/`Dihedral` in degrees. Failed or incomplete steps are not silently promoted to converged.

## Scope limitations

- Golden coverage includes a specific ORCA version, input format and small example molecules. Other datasets, advanced Scan grids, TD-DFT root-following and all proprietary formats require further tests.
- The output used for `Gibbs` and `Spin` is parser-read numerical data, not independently reproduced quantum-chemical accuracy.
- Neither an actual Windows ORCA/MPI execution nor the original v1.5.1 GUI/update/rollback real-device gate has yet been completed.
- The v1.6.0-rc.1 published tag remains frozen; changes on this development branch must go through review and a separately versioned release.
