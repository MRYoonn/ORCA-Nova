# ORCA Nova · Windows 연구실 PC 수용 시험 (P0)

## 자동으로 확인한 범위와 미검증 범위

GitHub Actions의 Windows 빌드/Go 테스트는 **ORCA 바이너리가 설치되지 않은 테스트 환경**에서 진행합니다. 이 테스트는 2~4개 mock Job의 병렬 실행·자원 예약·개별 중단·자식 프로세스 트리 정리·파일 격리·보고서 보존·업데이트 롤백을 확인하지만, 실제 ORCA/MPI 계산의 성공을 증명하지 않습니다.

**실제 연구실 PC ORCA 6.x 검증 및 안정판 승격은 사용자 실행 증거 없이 완료 처리하면 안 됩니다.** 이 파일은 그 증거를 안전하게 수집하기 위한 절차입니다. 실제 연구용 분자 구조나 계산 결과를 공개 GitHub에 업로드하지 마세요.

## A. 실제 실행

1. 기존 버전을 덮어쓰지 말고 새 폴더에 최신 Portable ZIP 전체를 풉니다.
2. `ORCA_Nova.exe`를 실행하고 **범고래 작업 표시줄/창 제목 아이콘** 및 UI 텍스트/배율을 눈으로 확인합니다.
3. `orca.exe`의 절대 경로와 결과 작업 폴더를 설정합니다. 해당 ORCA 6.x 설치·MPI 설정의 사용 권한은 본인 환경에서 확인합니다.
4. 미발표 연구용 구조 대신 공개 검증 가능한 간단한 XYZ로 먼저 `A4 S₀ Opt + Freq` 실행 후 OUT, GBW, 최적화 XYZ, Frequency 결과를 확인합니다.
5. 동일 분자에서 `W2 S₀ Opt → T₁ UKS Opt + Spin Density`를 실행하여 **T₁ 최적화·spin population·cube(.cube/.cub)** 결과를 확인합니다. `⟨S²⟩`, imaginary modes와 방법/용매를 수동 검토합니다.
6. 2개 및 4개 Job을 분리된 폴더에서 실행해 **모든 실행 중 Job의 예약 CPU/RAM이 예산을 초과하지 않는지**, 작업별 INP/OUT/GBW가 섞이지 않는지 확인합니다. 4개 동시 시작이 가능한 PC 자원이라도 큰 TD-DFT 단계는 동시 실행 1개가 안전할 수 있습니다.
7. 1개 Job만 취소합니다. 다른 계산이 그대로 진행되는지, `orca.exe`/MPI 자식 프로세스가 남지 않는지 Windows 작업 관리자로 확인합니다. 프로그램 종료/재실행, 대기 작업 보류 및 재시도 시 원본 데이터가 보존되는지 확인합니다.
8. 실행 중 업데이트 설치가 차단되는지 검사합니다. 완료 후 승인된 **안정판**의 업데이트 검색/설치/재실행/롤백과 기존 설정 보존을 검사합니다. 시험판은 안정판 자동 업데이트 채널에 강제 삽입하지 않습니다.

## B. 계산 폴더 자동 증거 검사

PowerShell을 프로젝트 폴더에서 열고 다음과 같이 실행합니다(실제 경로로 변경).

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tests\windows-orca6-smoke-check.ps1 -OrcaExe "C:\ORCA\orca.exe" -JobFolder "D:\ORCA_Nova\jobs\my_calculation" -RequireBothStates -RequireCube -RequireFrequency -RequireGBW
```

이 스크립트는 **기존 계산 결과의 증거만 검사**합니다. ORCA 계산을 시작하지 않으며 실제 설치 경로와 데이터는 PC 내부에만 남습니다. 각 INP에 대응하는 OUT의 `ORCA TERMINATED NORMALLY`, `SCF CONVERGED`, Opt/Freq 관련 출력, 계산별 GBW 및 XYZ, Triplet/Singlet Multiplicity, 최소 1개 Cube를 확인합니다.

스크립트가 기록하는 `Windows_ORCA6_Smoke_Audit.json`에 `Passed: true`가 나오더라도 이것은 화학적 정확성 및 원자별 Spin 해석의 증거가 아닙니다. 사용자 눈으로 확인해야 하는 아이콘·프로세스 종료·업데이트 항목은 `HumanVisualChecks`에 남습니다.

## C. 안정판 배포 게이트

- Windows EXE / WebView2 실제 실행 및 아이콘: 확인됨
- 실제 ORCA 6.x S₀ Opt+Freq, T₁ Opt+Spin Density, CUBE/GBW: 확인됨
- 2~4개 Job 예산·취소·자식 ORCA/MPI 종료: 확인됨
- 합계 CSV/TXT, 결과의 kcal/mol/eV/Eh 구분과 누락 항목: 확인됨
- 승인형 업데이트/재실행/복구 및 사용자 설정 보존: 확인됨
- 미공개 연구 데이터 GitHub 업로드 없음: 확인됨

모든 항목이 실제 환경에서 확인되기 전에는 릴리스 채널을 안정판으로 변경하지 않습니다.
