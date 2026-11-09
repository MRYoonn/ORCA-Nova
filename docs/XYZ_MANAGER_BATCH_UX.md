# XYZ Manager + Multi-XYZ Parallel UX — 2026-11-09

## Scope and safety
This is a frontend usability change. It does **not** change the ORCA queue scheduler, change existing on-disk ORCA calculation files, or delete original uploaded XYZ files from Windows.

The local molecule editor and Batch registration list are separate:
- **Basic XYZ Delete** removes only the primary in-memory source. The generated INP preview resets, and running a new single-molecule calculation requires explicitly choosing a new basic XYZ. Additional XYZ are never silently promoted after deletion.
- **Additional XYZ Delete** is per-file. Each registered extra shows a name and a Delete button; selecting the same file again after deletion is allowed.
- **Clear all input XYZ** resets the primary and extras only, with confirmation. Already submitted Queue jobs and saved results are unchanged.
- **Send all XYZ to Batch** copies basic + extras into Batch registrations; duplicate names in the Batch list are skipped. The original single-molecule inputs stay available.
- **Batch registrations** have separate checkboxes for `선택` (bulk remove) and `실행` (include in next submitted Job). Existing pending/running/completed Queue entries are never deleted by the registration controls.
- Bulk controls: select all/deselect all, remove selected, clear registration list; destructive list-only changes require confirmation.
- Parallel shortcuts: `동시 2개` and `동시 4개` adjust concurrency **only**. Scheduler still enforces configured CPU and RAM budgets.
- A budget hint reads `%pal nprocs`, `PALn`, and `%maxcore` from the current captured INP, and estimates the conservative requirement `CPU × MaxCore × 1.25` per Job. This is an estimate, not a hard RSS cap. Do not increase CPU/RAM budgets above the machine's actual resources just to force concurrent runs.

## Typical example for a 16-core / 64GB machine
For **two simultaneous** jobs of `nprocs 8` and `%maxcore 2500`, conservative reserved memory is ~25,000 MB/job, or ~50,000 MB for two. A total CPU budget of 16 and a RAM budget around 50,000 MB can admit two jobs, while leaving physical RAM for the OS. These are example calculations; actual free memory, other apps and ORCA MPI behavior must be checked.

Two 16-core jobs with `%maxcore 2800` could reserve roughly 112,000 MB combined, far above 64 GB. The Queue should **wait**, not overcommit.

## Verification
- Browser QA: `node tests/batch-ui.cjs` (Chromium/Playwright). Checks four independent XYZ workflow requests, concurrency 2 queue submission, exclusion, per-row removal, bulk INP import, drag/drop and results comparison; additionally checks primary/extra deletions, in-memory transfer to Batch, multi-selection and bulk removal, no mutation of previously submitted jobs.
- Syntax: `node tests/scientific-ui-syntax.cjs`.
- Windows CI: Go tests/vet, WebView2 host and Portable package, all unchanged.
- The tests do **not** directly operate real ORCA/MPI; live machine scientific validation remains a separate release gate.

## Useful next steps (not included)
1. Per-molecule dropdown presets in the Batch table, instead of editing the INP manually for different methods.
2. Lightweight 3D XYZ preview with atom index overlay and safe C5/C6 assignment assistance.
3. A live local CPU/RAM probe that proposes safe concurrency and per-Job core budgets without exceeding actual machine resources.
4. Save/restore the unsubmitted Batch registration list as a local manifest, with per-molecule custom INP edits retained.
5. One-click retry of selected failed jobs with isolated new folders and safeguards for previous GBW/HESS artifacts.

**Release:** do not overwrite the existing `v1.6.0-rc.2` tag or present this feature as already included in the public RC. A new Release is required for user installation.
