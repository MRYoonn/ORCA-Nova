# HAT / NEB-TS + Gibbs + IRC (D9)

This is a general two-endpoint intramolecular H-transfer workflow. The neutral doublet preset is a starting point, not a demonstrated reaction mechanism or a benchmarked method.

## Using the preset
1. In Avogadro, duplicate the reactant geometry and move the same H atom to the acceptor. Keep atom identity/order and composition unchanged. XYZ element-order matching alone cannot distinguish swapped atoms of the same element.
2. Import the reactant as the primary XYZ. Add the other geometry as `product.xyz` using the additional XYZ picker. Do not name the primary geometry product.xyz.
3. Select **1,3-HAT · NEB-TS + Gibbs + IRC** under the recommended presets, or D9 in the reaction mechanism library.
4. Review charge, multiplicity, solvent, method, temperature and resources. Defaults: neutral doublet, UKS M06-2X/def2-TZVP (ORCA input keyword: `M062X`), RIJCOSX, CPCM(DMF), TightSCF; no extra empirical dispersion by default. Use the same calculation level on both endpoints and the TS. The solvent can be changed to match the experiment.
5. Run the entire desktop workflow (not independent Batch jobs).

The stages are independent reactant and product Opt/Freq, NEB-TS/Freq using both optimized geometries, and IRC from the resulting TS. A manually supplied TS guess is unnecessary. The first three stages must have a converged SCF/geometry, final Gibbs energy, and respectively 0, 0, 1 imaginary modes. An imaginary mode count alone does not establish the mechanism. Inspect the mode animation and both IRC endpoints, spin populations, spin contamination, and plausible conformers. In particular, the fused-ring HAT geometry must have the intended stereochemistry.

The report provides reaction free energy G(P)-G(R), forward barrier G(TS)-G(R), and reverse barrier G(TS)-G(P), converted from Eh to kcal/mol, when stationary-point checks pass. These values remain provisional until the actual HAT mode and IRC connectivity are confirmed.

A ZIP export is available for manual use; its CMD is not equivalent to desktop scientific validation or automatic Gibbs reporting. A single workflow takes two endpoints and is intentionally unavailable in independent molecule Batch mode. Solvent-assisted HAT is possible to model by adding the same explicit solvent molecules to both XYZ files (not an automatic solvent-placement feature).

References: [ORCA 6.1 NEB-TS tutorial](https://www.faccts.de/docs/orca/6.1/tutorials/react/nebts.html), [IRC manual](https://www.faccts.de/docs/orca/6.1/manual/contents/structurereactivity/irc.html).

Validation status: source tests cover input/dependency failures and missing report data; actual ORCA/Windows NEB-TS execution and endpoint validation still require a real calculation.
