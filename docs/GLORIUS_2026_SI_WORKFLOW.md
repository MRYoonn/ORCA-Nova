# Glorius 2026 SI-level presets

## Evidence and scope

Hartmann, Liljenberg, Domack, Hölter, Daniliuc and Glorius, *Energy-Transfer Catalysis Enables the Birch-type Reduction of 2-Pyridones*, Angew. Chem. Int. Ed. 65 (2026), e9565676. DOI: https://doi.org/10.1002/anie.9565676. Source: Supplementary Information S48–S50 (section 4, Table S4, Figure S16/S17). No private research structures or copyrighted PDF files are distributed here.

The SI states ORCA 6.0.1; RDKit initial conformer generation; GOAT/GFN2-xTB conformer search; M06-2X/D3zero/def2-TZVP optimization, frequencies and IRC; higher-level def2-QZVPPD electronic energies; def2/J, RIJCOSX and CPCM(acetonitrile) for DFT. It validates minima with zero imaginary frequencies and TS with one imaginary frequency, visualization and IRC. The composite energy combines high-level electronic energies and low-level harmonic Gibbs corrections. Figure S16 reports Mulliken spin population.

**Reconstruction, not verbatim author input.** The data record https://doi.org/10.19061/iochem-bd-6-619 has the original calculations, but the original INP archive was not inspected. The supplied second SI ZIP was crystallographic (CIF), not an ORCA input archive. SCF thresholds, frequency temperature, Hessian initialization, xTB solvation and exact GOAT parameters remain distinct from author-confirmed settings. App defaults: TightSCF (editable), frequency temperature 298.15 K (editable), TS Calc_Hess true (see ORCA manual), no explicit xTB solvation. The DFT model is fixed in the Glorius calculation group; use general presets or edit an INP for a different model.

## Inventory

| ID | Purpose | Inputs / outputs |
|---|---|---|
| G1 | GOAT | XYZ + selected charge/multiplicity; GFN2-XTB minimum and ensemble |
| G2 | S0 Opt + Freq | Singlet, minimum |
| G3 | T1 Opt + Freq | UKS triplet, minimum, Mulliken |
| G4 | Doublet Opt + Freq | UKS doublet, minimum, Mulliken |
| G5 | Triplet TS | TS guess, triplet, OptTS/Freq |
| G6 | Doublet TS | TS guess, doublet, OptTS/Freq |
| G7 | IRC | Verified TS geometry and matching electronic state |
| G8 | High-level SP | optimized XYZ, matching electronic state, def2-QZVPPD |
| G9 | Mulliken SP | open-shell optimized XYZ; convenience template, independent SP not documented |
| G10 | GOAT -> Opt/Freq -> SP | same electronic state, composite Gibbs |
| G11 | GOAT -> S0/T1 Opt/Freq -> each SP | shared singlet GOAT initial conformer, independent relaxations, two composite values, approximate comparative gap |
| G12 | TS Opt/Freq -> high SP -> IRC | user-provided TS guess; no unconstrained GOAT on guessed transition state |
| G13 | Opt/Freq -> SP | selected conformer, skip GOAT |

G10/G11/G12/G13 use the desktop/Batch executor and automatically write combined Gibbs values to analysis.txt and results.json. The exported standalone CMD script only executes stages and checks basic termination; use the desktop executor for stationary validation and Gibbs postprocessing. G11 uses a shared conformer as an application convenience; not a full independent conformer search for every state.

## Validation

- GOAT accepts ORCA termination, global minimum and ensemble markers plus actual output XYZ with matching atom number, composition and order. Uses basename.globalminimum.xyz, not basename.xyz; refuses DFT launch if missing.
- For the declared DFT minimum/TS stages, require converged SCF, optimization, energy, harmonic Gibbs and 0/1 imaginary modes respectively. Visualize the TS imaginary mode and IRC endpoints before interpreting a chemical transition.
- High-level SP must consume the named earlier stage's optimized XYZ and matching charge/multiplicity. SI-level Gibbs is calculated from actual output and input file data, never filled in from expected values. Results with missing/failed output stay unavailable.
- Composite T1-S0 comparison is limited to same-source, equal-charge, matching atom-count singlet/triplet minimum pairs; confirm chemical identity, temperature and electronic-state identity. Not a TD-DFT vertical excitation energy, nor a guaranteed lowest T1 assignment.
- The app does **not** automatically optimize all GOAT ensemble members, generate RDKit conformers from SMILES, establish TS connectivity, calculate general stoichiometrically balanced reaction barriers, or apply an undocumented quasi-harmonic/standard-state correction. GOAT requires ORCA 6 with working xTB.
- Windows CI tests with synthetic outputs and helper executable are not real ORCA calculations; original ORCA 6.0.1/xTB/DFT scientific reproduction remains to be tested.

References: ORCA 6.0 GOAT manual https://www.faccts.de/docs/orca/6.0/manual/contents/typical/GOAT.html ; XTB keyword syntax https://www.faccts.de/docs/orca/6.0/manual/contents/detailed/model.html ; TS/IRC https://www.faccts.de/docs/orca/6.0/manual/contents/typical/optimizations.html .
