# Batch Calculation development

This branch adds Multi-XYZ / INP import, per-molecule editable requests, a persistent resource-aware queue, strict stage validation, per-job reports, batch exports, and a comparison table. It is development code, not a scientifically validated stable release.

## Run and test

Use Go 1.23, `go test -race -count=1 ./...`, `go vet ./...`, and `go build .`. No third-party Go dependencies are required. Queue tests launch the test executable as a cross-platform Mock ORCA; these do not execute licensed ORCA.

Browser tests require Node and Playwright (install in a separate tools directory). From the checkout run `NODE_PATH=/path/to/tools/node_modules CHROMIUM_PATH=/usr/bin/chromium node tests/batch-ui.cjs`. Omit CHROMIUM_PATH to use Playwright's installed browser. This suite uses HTTP API fixtures; the current development run additionally exercised the real Go HTTP server with Mock ORCA and four XYZ molecules.

Windows Portable Build now runs for `feat/**` pushes, pull requests, and main. It tests the backend, cross-target builds on Windows, embeds icons, builds the .NET Framework 4.8 WebView2 host and packages the complete portable ZIP. A Linux-produced backend EXE alone is not the complete portable application. Do not create a stable Release before real Windows/ORCA validation.

## Workflow

1. Save the absolute ORCA executable path and work folder in the existing runner.
2. Choose the existing preset and CPU/MaxCore settings. Import XYZ files together or use Drag & Drop. Imported INPs require their initial XYZ; dependent INPs are ordered by their geometry dependencies.
3. Adjust each molecule's Charge/Multiplicity or expand and edit its INPs. Explicitly applying the preset regenerates edited inputs after confirmation. Exclude or delete unwanted molecules.
4. Set concurrency (default 1), total CPU and RAM budgets, then run whole-batch Preflight. CPU cannot exceed detected logical CPUs. RAM is a user-selected budget, not a measurement of free system memory.
5. Start the queue. Independent molecules run concurrently when all reservations fit. Stages within each molecule run sequentially. A failure does not stop other molecules.
6. Use job controls to hold/reorder/cancel/retry. Pausing stops new dispatches. Stop All interrupts running stages and holds pending jobs. Pending cancellation is Cancelled; active cancellation is Interrupted. Cancel all Pending jobs before applying an update.
7. Each attempt has a fresh directory. Retry copies prior artifacts and revalidates completed stages; original directories remain untouched. Editing prior INPs on disk requires registering a new job, preventing stale resource reservations.
8. Queue state is beside the app configuration (`job_queue.json`). Restart always pauses dispatch and turns recorded Running entries into Interrupted. Corrupt/unwritable queue state blocks further dispatch and is not silently discarded.
9. Each job writes `analysis.txt`, `results.json` and legacy report names. Failures additionally write `FAILURE_ANALYSIS.txt`. `batch_results.csv` and `batch_summary.txt` in the work root aggregate the queue. The comparison table can export selected zero-based atom populations; ORCA indices are not chemical ring labels such as C5/C6.

## Resource and scientific safeguards

- A job reserves the largest stage CPU and largest stage `nprocs * maxcore * 1.25` MB throughout its workflow. `%maxcore` must be explicit. PAL keywords and inline `nprocs` are recognized. The reservation is a safety estimate, not a hard process-memory cap; leave RAM for the OS and other applications.
- Unsupported `%base`, external orbital and compound input directives are rejected for queue jobs. Duplicate stage inputs and imported geometries colliding with optimization outputs are rejected. External/custom workflows require separate support, rather than unsafe rewriting.
- Successful stages require normal ORCA termination, SCF convergence, finite Electronic Energy and, for Opt, convergence plus valid output XYZ with the same ordered elements. Explicit Charge/Multiplicity output mismatches are rejected. This does not prove excited-state identity or Root Following success.
- Missing data remains Not calculated / Unavailable / Failed. Electronic, Gibbs and Vertical excitation energies are distinct. Relative/Adiabatic S1/T1 energies are not inferred from an arbitrary pair of outputs. Comparison includes Method and Geometry. Spin Population alone does not establish nucleophilicity/selectivity.
- ETA uses prior equivalent successful stages and returns a range. Missing history gives 추정 중; overruns give 예측 불안정. Batch ETA is a conservative range, accounting for possible serial execution, not a guaranteed finish time.
- Updates block both legacy execution and queue admission/dispatch once installation starts. Windows cancellation uses taskkill /T /F; Linux uses a process group. Actual Windows ORCA/MPI child cleanup and recovery after abrupt OS/process termination still need device tests.

## Remaining validation / external blockers

The development machine is Linux without Windows/.NET Framework/WebView2 or an ORCA installation. Windows native UI, MPI cleanup, scientific S1/T1 state tracking and actual ORCA 6.x outputs must be tested on Windows. Mock output is not scientific validation.

During this run Git HTTPS reads worked, but HTTPS requests to `api.github.com` and `api.crossref.org` received proxy CONNECT 403. GitHub PR/Actions API operations require the former domain in environment network settings and usable scoped GitHub API authentication. Literature metadata/SI research requires the latter plus the publisher/SI hosts identified from real papers. No token values should be supplied in chat.

No Glorius SI-backed template was added: title, DOI, SI page and actual computational settings could not be verified. Existing general presets are not relabeled as literature methods. Obtain the public SI, record its exact title/DOI/page/program/settings, and validate ORCA compatibility before adding any template; do not automatically translate Gaussian inputs.

For the real HTTP/browser Mock scenario on Linux with Python 3: `go build -o bin/orca-nova .`, then `NODE_PATH=/path/to/tools/node_modules CHROMIUM_PATH=/usr/bin/chromium node tests/batch-http.cjs`. It starts an isolated server with a temporary configuration and work folder, exercises the actual browser UI/queue API and prints the retained evidence directory. Set `ORCA_NOVA_BIN` to use another backend build. The checked-in `tests/mock_orca.py` is deliberately synthetic; never use it for chemistry.
