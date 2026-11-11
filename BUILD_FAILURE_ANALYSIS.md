# Build failure analysis — 2026-11-08

- Workflow run: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42474368596
- Job: Windows Portable Build (`128712263438`)
- Result: Go test PASS; Go Windows EXE build PASS; native WebView2 host build FAIL.
- Actual error: `native_host/Program.cs(3,18): error CS0234: The type or namespace name 'Http' does not exist in the namespace 'System.Net'`.
- Root cause: the .NET Framework 4.8 C# project did not explicitly reference `System.Net.Http`.
- Fix: add `<Reference Include="System.Net.Http" />` to the WinForms host `.csproj`.
- Additional preventive fix: package only the native host EXE, DLLs and `.config` files, preserving runtime subdirectories. This matches the update installer's accepted-file whitelist and excludes `.pdb` / `.xml` debugging artifacts.
- Re-test: Windows Actions rerun required; actual ORCA calculation and taskbar pinning still need Windows on-device checks.

## 2026-11-12 — Triplet Geometry Transfer branch (NOT released)

- Branch: feat/triplet-geometry-transfer-20261112; PR [#50](https://github.com/MRYoonn/ORCA-Nova/pull/50).
- Initial run [43048580010](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43048580010): Go build undefined tripJobMeta/writeTripletSummary/validateTripletStage due to incomplete multi-file feature commit. Fixed by adding triplet metadata and triplet_results.go.
- Run [43050804438](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43050804438): queue_test.go:104:33 expected statement, found else in added mock ORCA fixture. Fixed Go } else { syntax; commit 6508b6e4.
- Runs [43052568333](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43052568333) and [43052559484](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43052559484): same-target mock test failed after adding STEP3 GBW MOREAD because method fingerprint incorrectly treated initial guess keyword as a new chemical method. Fixed by excluding MOREAD/NOAUTOSTART from Method fingerprint and adding internal GBW-only allowlist tests. Full Windows Go tests/EXE package smoke later passed [43052750509](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43052750509).
- **Scientific release gate still open:** Actual licensed Windows ORCA6/MPI runs for S-alkyl thiouracil and resulting verified OUT, XYZ, GBW, CUBE, ⟨S²⟩, spin populations, Frequency have not been executed in CI. Do not promote to Stable until these pass and real chemical-state limitations are reviewed. Current latest stable is v1.6.1.
