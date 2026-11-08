# Golden / scientific release review

Review date: 2026-11-09 Asia/Seoul. Base reviewed: PR17 head `cb6bc66c43becccdafddabf2f79cb4d49b7740f3`.

## Real source retrieval

Fresh HEAD/redirect probes to the requested article DOI `https://doi.org/10.1002/anie.9565676`, dataset DOI `https://doi.org/10.19061/iochem-bd-6-619`, and previously located PMC supplement `https://pmc.ncbi.nlm.nih.gov/articles/instance/13427111/bin/ANIE-65-e9565676-s002.zip` all failed with proxy CONNECT HTTP 403. No content was downloaded. The article/SI/dataset relationship and redistribution license were therefore not independently reverified in this run. Prior documentation describes Figure 5 as verified in an earlier environment, but that is not new SI or ORCA data evidence.

No actual ORCA Golden OUT/INP fixtures exist in the reviewed checkout. Current scientific and visualization tests construct synthetic strings. They must not satisfy the release gate requiring real ORCA outputs. No user-private molecule data was copied or published. Resume by obtaining permitted access to the DOI and identified publisher/dataset hosts, inspect provenance and license, then extract only permitted minimal fixtures and independently read expected values.

## Concrete scientific guards corrected

- Keyword-only comparison ignored `%cpcm` and other blocks. Profiles now additionally require conservative exact agreement of non-keyword input settings. This can reject benign resource/block differences and requests review rather than hiding a scientific mismatch.
- TD-DFT/IRoot inputs are rejected for automatic UKS Adiabatic / ground-state Reaction Profiles; stripping `tddft` from Method was insufficient to establish state identity.
- Missing XYZ no longer bypasses composition checks merely because the user checked confirmation. Registered source XYZ must be readable and its elemental formula validated for every role.
- Available imaginary modes now reject a claimed Minimum even for Electronic comparisons; TS still requires exactly one mode. Missing Frequency does not become a measured zero.
- Existing analysis warnings and explicit <S²> deviation warnings accompany profile results.

Regression tests cover differing solvent blocks, excited-state inputs, missing XYZ, and matched settings. Synthetic fixtures were updated to include explicit geometry for cross-job tests. Real ORCA/MPI execution, scientific Golden tests and full SI reproduction remain NOT RUN and continue blocking stable release.
