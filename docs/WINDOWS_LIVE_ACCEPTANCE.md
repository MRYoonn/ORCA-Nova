# Windows Portable Live Acceptance / Real ORCA Release Gate

## Verified CI run

The GitHub Actions workflow `Windows Published Portable Live Acceptance` downloads the published **v1.6.0-rc.2** ZIP and SHA256SUMS.txt, verifies the SHA-256 digest, expands the *actual released* Portable archive, launches `ORCA_Nova.exe` on a Windows runner, and checks version, protected localhost API and a water Preflight request.

Passing example: https://github.com/MRYoonn/ORCA-Nova/actions/runs/42577561314

This closes the **Windows packaged backend startup / local API** portion of the release gate. It **does not** test the physical user's Windows GUI, a licensed installed ORCA executable, OpenMPI, process cancellation or live auto-update through the stable release channel.

## Safe local test (Windows PowerShell 5.1+)

Download the published assets separately:

- https://github.com/MRYoonn/ORCA-Nova/releases/download/v1.6.0-rc.2/ORCA_Nova_Portable_Windows.zip
- https://github.com/MRYoonn/ORCA-Nova/releases/download/v1.6.0-rc.2/SHA256SUMS.txt

Run this command from the project checkout:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tests\Windows-Portable-LiveAcceptance.ps1 -PortableZip ".\ORCA_Nova_Portable_Windows.zip" -ChecksumFile ".\SHA256SUMS.txt" -ExpectedVersion "1.6.0-rc.2"
```

Optional **actual licensed ORCA 6.x / Windows MPI** H₂O run:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tests\Windows-Portable-LiveAcceptance.ps1 -PortableZip ".\ORCA_Nova_Portable_Windows.zip" -ChecksumFile ".\SHA256SUMS.txt" -ExpectedVersion "1.6.0-rc.2" -OrcaExe "C:\Program Files\ORCA\orca.exe" -RealOrcaTimeoutSeconds 300
```

Replace the example ORCA path with the actual `orca.exe` location on the machine. The test explicitly sets `%pal nprocs 2` to exercise a two-core ORCA launch through the actual frontend's `/api/run`. It checks completed status and that the output contains `ORCA TERMINATED NORMALLY` and `FINAL SINGLE POINT ENERGY`.

The test writes `ORCA_NOVA_WINDOWS_ACCEPTANCE.json` without the local API secret. It uses a unique temporary folder and isolated `APPDATA` setting, does not touch the real ORCA Nova preferences/jobs, and deletes its temporary test data by default. Use `-KeepTemporaryFiles` only if diagnosing a failure. Do not publish private or unpublished molecular outputs.

## Manual Windows GUI, update and cancellation acceptance checklist

1. Unzip the RC into a **new empty folder** and launch the normal `ORCA_Nova.exe` (not test mode). Confirm the WebView2 native window opens, menu/buttons are usable and the results graphs render. The CI test skips the GUI.
2. Enter the installed ORCA 6.x `orca.exe` path in Settings; make a new H₂O test job. Confirm live OUT logging, report generation and the completed final energy. The automated optional test can cover this.
3. Use a separate, deliberately slow **public, nonprivate** calculation to test Cancel. Confirm both the ORCA and its MPI subprocesses stop, the job is marked interrupted, and other jobs remain usable. Never kill the user's unrelated scientific processes.
4. Back up real v1.5.1 settings and jobs. Confirm the stable release does **not** offer v1.6.0-rc.2 as an automatic update, because the RC is a Pre-release.
5. Before any future stable v1.6.0, test its actual update from an isolated v1.5.1 install and verify the new native UI relaunch, settings preservation and forced-failure rollback. Do not force prereleases into the stable channel to fake this gate.

### Release decision

Verified: published ZIP checksum, actual packaged EXE boot + session API + Preflight, the real ORCA 6.1.1 parser Golden fixtures, Windows PowerShell transactional rollback tests.

Still unverified: user-machine native WebView2 interactions, real local ORCA/MPI start/stop, live v1.5.1 → future stable update (including native UI relaunch and power-loss recovery). **Keep v1.6.0 as a prerelease until these gates are completed.**
