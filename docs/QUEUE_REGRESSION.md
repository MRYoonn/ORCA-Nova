# Queue and process cancellation regression audit

Base: PR #12 head `e54d82ef7776bfa5c9f1cdcf7f876320b67c022f`. The existing batch implementation was retained. It already contains failed-stage GBW/HESS isolation, immutable original XYZ/INP checks, completed-stage reuse, resource reservations, recovery and four-job failure/cancellation tests.

## Corrections

- Retry is now transactional for in-memory attempt metadata. If artifact copying or queue persistence fails after creating a new attempt folder, the original Job pointer, Failed/Interrupted state and held flag are restored. Original results are never deleted. A newly created incomplete retry directory may remain for diagnosis.
- Artifact copying now accepts only regular files. Directories, symlinks and special files such as FIFOs cannot be copied as completed-stage artifacts, avoiding blocking reads from special files.

## Validation performed

`go test -race -run 'TestCancellationTerminates|TestRetrySaveFailure|TestRetryDoesNotCopy|TestBatchFour' -count=1 ./...` passed on Linux (8.661 seconds).

- A real helper process starts a descendant heartbeat process. Cancellation stops both, verified by the heartbeat ceasing to change. This portable test uses the same Windows `taskkill /T /F` and Unix process-group implementations as the application. Windows execution remains pending CI at the time this note was written.
- A forced queue-save error during retry restores original metadata and exposes the persistence error while pausing dispatch.
- Existing tests verify failed-stage GBW/HESS are absent from retry directories, completed artifacts survive, old result directories remain, and four independent jobs continue around failure/cancellation/retry.

These are Mock ORCA tests, not licensed ORCA/MPI scientific validation. Abrupt OS termination and Windows ORCA/MPI behavior still require device tests. No updater code changes were made by this audit; updater coverage is maintained separately by the integration task.
