# Scientific profile and Glorius reference-method validation

Date: 2026-11-09. Branch: feat/scientific-profiles-scan-spin (stacked on PR #15, then #12).

## Added calculations (based only on actual .out/.inp files)

- `/api/visualization/profile` POST, authenticated: user assigns S0/T1 or Reactant/TS/Product to distinct OUT files from one existing job folder. No inferred filenames or fictitious energies.
- All outputs must have normal termination, converged SCF, optimized geometry, INP method/geometry metadata, matching method/basis/dispersion/solvent/charge and compatible spin states. Gibbs comparisons additionally require observed temperatures and frequency results; TS needs exactly one imaginary frequency.
- The user confirms stoichiometry and chemical identities before calculation. Units: relative kJ/mol, raw Eh also retained. This is a 3-structure profile, **not** a multi-species stoichiometric sum and not an automated IRC/TS identity proof.
- The S0/T1 value is computed as the energy of optimized UKS triplet minus optimized singlet, not TD-DFT vertical root energy. Root identity and spin contamination require human review.
- Step-resolved spin population parser recognizes Mulliken/Loewdin/Hirshfeld tables **only within each explicit Scan Step block**; missing data stays unavailable, never copied from the final step.
- The existing GUI now exposes selected source mapping, opt-in identity confirmation, plot/table exports and explicit error messages. No new layout or 3D engine.

## Glorius 2026 published method reference

Source 1: Hartmann, Liljenberg, Domack, Hölter, Daniliuc, Glorius, *Energy-Transfer Catalysis Enables the Birch-Type Reduction of 2-Pyridones*, Angewandte Chemie International Edition 65 (2026), e9565676, DOI https://doi.org/10.1002/anie.9565676.

The published **article's Figure 5 caption** specifies M06-2X(D3Zero)/def2-QZVPPD/CPCM(MeCN) // M06-2X(D3Zero)/def2-TZVP/CPCM(MeCN), with relative Gibbs free energies. Figure 5 is not evidence that the user's molecule has the same chemical state or thermochemistry.

- Preset A = `! M06-2X def2-TZVP D3ZERO TightSCF Opt Freq CPCM(acetonitrile)` (general single-input template matching published optimization level).
- Preset B = `! M06-2X def2-QZVPPD D3ZERO TightSCF CPCM(acetonitrile)` (single-point calculation using the **optimized geometry from A**).
- The final corrected Gibbs free energy needs the explicitly validated high-level electronic energy + appropriate thermal corrections, appropriate standard-state and conditions. **Preset B alone does not produce the reported composite Gibbs energies.**
- Input electron configuration, correct charge/multiplicity, geometry and reaction species are always user decisions.
- `D3ZERO` and `def2-QZVPPD` keywords are present in the ORCA 6.1 official documentation:
  https://www.faccts.de/docs/orca/6.1/manual/contents/modelchemistries/dispersioncorrections.html
  https://www.faccts.de/docs/orca/6.1/manual/contents/essentialelements/basisset.html

**Verification level:** Published main text Figure 5 + official ORCA manual verified. Full supplementary PDF (ANIE-65-e9565676-s001.pdf) could not be inspected from the available environment; publication-specific input files/thermochemical conventions and actual ORCA runs remain unverified. The UI says `문헌 Method`, not `SI 완전 재현`.

Source 2 (candidate for future paper-specific SI comparison): Hümpel, Rana et al., Glorius, *Accessing Medium-Sized Bridged Heterocycles via EnT-Catalyzed Intermolecular Dearomative (5 + 4) Cycloaddition of Furans and Oxazoles*, JACS (2026), DOI https://doi.org/10.1021/jacs.6c00671. The journal lists SI Section 4 (Computational Calculations) but its detailed method has not been verified here; **no invented preset**.

## Validation boundaries and next evidence

- Unit regression tests: correct sign/unit for adiabatic gap and reaction barrier; fail closed for unconfirmed identity, mismatched method, missing Gibbs/frequency and wrong spin state. Explicit Scan-step partial data test; ORCA dotted temperature parsing test.
- Windows Actions compile/Go tests/Vet/package, and browser checks from prior PR #15 provide software confidence, **not scientific accuracy**.
- This environment has no installed licensed ORCA executable or real Windows GUI/MPI installation and cannot prove process cleanup on the user's workstation, real state identity, or seamless updater installation/rollback.
- Must inspect real ORCA 6.0/6.1 OUT/INP from trusted nonprivate test compounds before greenlighting stable release. Do not upload user's unpublished XYZ or calculations to public repository.
