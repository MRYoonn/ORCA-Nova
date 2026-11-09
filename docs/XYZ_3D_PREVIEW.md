# XYZ 3D Structure Preview (development branch)

## Scope

- Preview the primary XYZ, any independently registered additional XYZ, or a molecule in the Batch registration list. The viewer uses **in-memory** text only; no web assets or remote services are required.
- Orbit with pointer drag, zoom with wheel, click any atom to see the element, its XYZ coordinates (Å), XYZ line number and **ORCA atom index, zero-based**.
- Toggle atom index labels, reset camera, and copy the selected index when clipboard access is permitted.
- Optionally mark selected atoms as C5 or C6; **these labels are manual, temporary, unverified chemical assignments**. The app does not infer uracil numbering from XYZ atom order.
- The viewer draws covalent-distance candidate bonds as a visual convenience. **It is not a bond-order, tautomer, stereochemistry, radical, charge, or spin-density calculation.** Metals, unusual bonding, and stretched geometries may display misleading or missing lines.
- XYZ parser validates atomic symbols, finite coordinates, declared atom count, <=800 atoms, and <=4 MB text. Invalid geometries are rejected for *preview*; the original input and pre-existing calculations are not modified. An input that passes preview must still pass normal Preflight.
- Removing the loaded XYZ registration still does not erase the original file, queued jobs, GBW or OUT results.

## QA

`node tests/scientific-ui-syntax.cjs` validates all embedded JavaScript scripts.

`node tests/xyz-3d-ui.cjs` exercises Chromium primary/extra/Batch view, water-geometry bond estimate, zero-based atom indexes, temporary C5/C6 marking, orbit/zoom, malformed inputs, and deletion isolation. The existing `node tests/batch-ui.cjs` guards the previous Queue functionality.

Workflow `.github/workflows/xyz-manager-browser.yml` now runs both browser regressions on this branch/PR. Windows packaging workflow independently checks Go and native WebView2 compilation.

## Release gate

Do not claim stable or published release status from source/CI alone. This feature is developed separately from published `v1.6.0-rc.3`. Check native WebView2 pointer/scroll/modal use on a real Windows 10/11 desktop and actual licensed ORCA/MPI before approving a stable release.
