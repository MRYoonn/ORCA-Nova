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

## 2026-11-12 — Triplet acceptance/Browser CI 후속 오류 분석 및 해결

1. **Windows PowerShell 구문 검사 실패** — [43054421169](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43054421169). 새 로컬 ORCA 수용시험 스크립트에서 `"$stem: ..."`이 PowerShell의 드라이브형 변수 구문으로 해석되어 ParserError가 발생. 모든 인스턴스를 `"${stem}: ..."`으로 수정(커밋 `819a2dd6`). [43054816015](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43054816015)에서 PowerShell 문법 재검사 성공.
2. **브라우저 테스트 환경 설치 실패** — [43053284032](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43053284032). `npx playwright install --with-deps chromium`에서 Ubuntu apt 패키지 미러 DNS 오류로 추가 fonts 파일 다운로드 실패. 애플리케이션 JS 버그가 아니므로 시스템 apt 의존성 설치를 생략한 `npx playwright install chromium`으로 변경(커밋 `3c96ab24`). Chromium 브라우저 실행과 전체 회귀 테스트 정상 확인.
3. **신규 Triplet 브라우저 QA 변수 범위 오류** — [43055506071](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43055506071). Playwright의 `page.waitForFunction`은 브라우저 문맥에서 실행되는데, Node 테스트 서버 변수 `queued`를 참조하여 `ReferenceError: queued is not defined` 발생. 브라우저에 실제 표시되는 Batch 등록 성공 문구를 대기하도록 변경(커밋 `ee791abc`).
4. **최종 재검증** — [Browser QA 43055771093](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43055771093): 신규 Triplet UI/매핑/2-Target Batch/Spin 비교, 기존 W2/W5·XYZ·SI·업데이트 등 브라우저 회귀 테스트 모두 통과. [Windows Portable 43055771569](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43055771569): Go 단위·Batch Mock subprocess·Go Vet·Windows EXE 빌드/인증 로컬 API Smoke·실행 전 원본 OUT 덮어쓰기 차단·실패 분석파일 테스트 통과.
5. **미검증 항목 (릴리스 차단)** — GitHub Actions에는 연구실 라이선스가 적용된 ORCA 6.x/MPI 엔진과 실제 thiouracil XYZ 입력이 없어, 실계산의 STEP1/STEP2/STEP3 최적화·GBW·CUBE·실제 Spin Population·Triplet 전자상태/허수 진동수 결과는 검증하지 못함. 해당 수용 시험은 `tests/Windows-Triplet-RealORCA-Acceptance.ps1`으로 로컬 수행하도록 준비. **v1.6.1 Stable 유지, PR #50 병합 및 Stable 배포 보류.**
