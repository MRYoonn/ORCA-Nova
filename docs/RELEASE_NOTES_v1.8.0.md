# ORCA Nova v1.8.0 — Stable

Windows Portable 정식 버전. 사용자 승인에 따라 검증된 v1.8.0-rc.2 제품 코드를 정식 채널로 승격합니다. 기존 v1.7.0 및 Beta 다운로드는 보존합니다.

## 새 기능
- Calculation / Batch / Results / Tools / Settings 독립 Workspace, 프리셋 검색, 간결한 설정.
- 전체 CPU 선택 및 RAM 자동 배분, 실행 중 Queue 추가, 미실행 Job 재배분, PC 전체 CPU/RAM 모니터. 직접 수정한 INP의 자원 설정 유지.
- Glorius 2026 SI 기반 13개 프리셋과 GOAT, Opt/Freq, TS/IRC, Mulliken, High-level SP, Composite Gibbs Workflow.
- Calculation 및 Batch의 결과 폴더 바로 열기, Results의 완료 작업 선택, Batch/Results의 경로 복사.
- 실행 버튼 근처 현재 단계/Workflow ETA, Batch 및 각 Job ETA. 유사 실행 기록 및 CPU/RAM/동시 실행 제한 기반 범위; 정보 부족 또는 초과 시 불확실 표시.
- Composite Gibbs는 검증을 통과한 단계와 일치하는 XYZ/전자 상태로 계산. 실행 실패·결과 누락 시 수치를 만들지 않음.

## 검증 범위
- [RC2 Windows](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44996147146), [Chromium](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44996147224), [Golden/Go](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44996147182), [RC2 versioned Windows release](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44996292281).
- 공개 ORCA 6.1.1 OUT의 SP, Opt/Freq, PES, TDDFT 및 triplet 파서 검증; Glorius 실행 흐름, Queue 및 재시작은 Mock 실행 검증.
- **실제 ORCA 6.0.1/6.x 및 xTB/MPI를 실행한 Glorius 전체 계산의 과학적 재현은 미검증입니다.** Stable은 앱 배포 채널을 의미하며 논문 수치 재현 인증이 아닙니다.
- SI의 계산 수준을 재구성한 것이며 저자 원본 INP는 미대조. SCF/온도/Hessian 등 앱 기본값은 원본 확인값이 아닙니다. GOAT ensemble 전체의 DFT 재평가, TS imaginary mode/IRC 종점, T1 전자상태 및 spin contamination은 별도로 확인하십시오.
- ETA는 근사치이며 MaxCore 예약량은 실제 RAM의 절대 상한이 아닙니다.
- [Glorius 상세 문서](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/GLORIUS_2026_SI_WORKFLOW.md)

## 설치 / 업데이트
기존 Settings에서 Stable 채널로 업데이트를 확인하거나 ZIP을 별도 폴더에 풀고 ORCA_Nova_UI.exe를 실행하십시오. 실행 중인 계산을 종료한 후 업데이트하고, 계산 폴더 및 사용자 설정을 별도로 백업하십시오. SHA256SUMS.txt로 ZIP 검증이 가능합니다. ORCA와 MPI/xTB는 별도 설치가 필요합니다.
