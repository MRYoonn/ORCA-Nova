# ORCA Nova 개발 인수인계

## 2026-11-22

- 브랜치 `feat/glorius-si-workflow`, [Draft PR #56](https://github.com/MRYoonn/ORCA-Nova/pull/56).
- 최신 제품 코드 커밋 [a1e773895c0b1dc859dd19134d3da907ca40d782](https://github.com/MRYoonn/ORCA-Nova/commit/a1e773895c0b1dc859dd19134d3da907ca40d782). 이 문서의 커밋은 `git rev-parse HEAD`로 확인한다.
- 공개 버전 [Stable v1.7.0](https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.7.0), [Beta v1.8.0-rc.1](https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.8.0-rc.1). `v1.8.0-rc.2` 릴리스 노트 준비 완료. 배포 승인과 병합 대기.

### 변경

- `glorius_workflow.go`, `preflight.go`, `main.go`, `report.go`에서 Glorius 검증 강화. GOAT/Opt/Freq/SP는 실제 `* xyzfile`과 실행 계획이 일치해야 한다. Opt/Freq로 선언된 단계에서 IRC와 상충하는 Opt/OptTS 조합을 차단한다. Composite Gibbs는 저수준과 고수준 단계가 이번 실행 또는 재시작 검증을 모두 통과한 경우 계산하며, 디스크 INP의 XYZ 참조를 보고서 생성 시 다시 확인한다.
- `eta.go`, `batch_summary.go`에서 단계 ETA와 Batch ETA가 동일한 기록 기반 시간 범위를 사용한다. 현재 단계에서 이미 경과한 시간을 범위 양 끝에서 뺀다. Batch 하한은 동시 실행 수, CPU, RAM 예약 한도에 근거하며, 상한은 직렬 실행을 가정한다. 자원 대기 및 재시도 시간은 별도로 변동한다.
- `assets/desktop.html`의 Batch와 Results에서 작업별 결과 폴더 경로를 복사할 수 있다. `tests/batch-ui.cjs`에서 클릭한 작업과 복사된 경로가 일치하는지 검증한다.
- `tests/batch-http.cjs`는 Batch Workspace로 이동하고 수동 자원 설정을 열도록 수정했다. 이전 테스트는 숨겨진 동시 실행 입력에서 대기하다 실패했다.

### 검증

- 로컬 Go 전체 회귀와 `go vet ./...` 통과. 공개 ORCA 6.1.1의 원본 blob을 확인한 뒤 SP, Opt/Freq, PES Scan, TDDFT, triplet 출력 5개 파서 검증 통과. Glorius의 GOAT→Opt/Freq→SP 실행 분기는 Mock ORCA로 검증했다.
- Chromium의 워크플로 지정 회귀 7개 및 JavaScript 구문 검사 통과. Chromium과 로컬 Go 서버를 연결한 4개 XYZ, 2개 병렬, 실패 후 재시작, 결과 TXT/JSON/CSV 내보내기 통과. 업데이트 UI는 로컬 Playwright 기본 브라우저 경로가 비어 있어 설치된 시스템 Chromium을 연결해 재검증했다.
- 로컬 Windows amd64 Go EXE와 테스트 바이너리 교차 빌드 통과. Windows WebView2와 Portable ZIP/EXE 실행 검증은 GitHub의 Windows CI에서 수행한다.
- 인수인계 당시 `8238886`의 [Windows](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44947636166), [Chromium](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44947636098), [Golden/Go](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44947636093) 성공 기록을 공개 페이지에서 확인했다. [변경 커밋의 CI](https://github.com/MRYoonn/ORCA-Nova/commit/a1e773895c0b1dc859dd19134d3da907ca40d782/checks)는 여기서 확인한다.

### 재개 지점

1. 최신 PR의 Windows, Chromium, Golden/Go 확인 후 승인된 범위에서 `.github/workflows/release.yml`로 Beta 배포한다. 후보 버전 `v1.8.0-rc.2`에 해당하는 공개 Portable ZIP, SHA256SUMS, 앱 버전 및 Beta 채널을 확인한다.
2. P1은 `queue.go`의 자동 자원 할당 및 Batch 재개 흐름에서 이어간다. 사용자 지정 자동 동시 실행 수의 효과와 ORCA/MPI 프로세스의 실제 CPU/RAM을 Windows에서 확인한다.
3. P2는 `docs/GLORIUS_2026_SI_WORKFLOW.md`에서 이어간다. 원본 INP, 실제 ORCA 6.x/xTB 실행, 독립 conformer 재순위화와 결과 분석이 남았다. Golden은 OUT 파서의 검증 범위다.
