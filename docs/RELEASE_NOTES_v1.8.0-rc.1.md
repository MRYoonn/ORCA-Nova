# ORCA Nova v1.8.0-rc.1 — Workspace & Automatic CPU/RAM

**Windows Portable 시험판 (Beta/Release Candidate). Stable v1.7.0 유지.**

## UI 개선
- Calculation, Batch / Queue, Results, Tools, Settings 독립 화면
- 추천 / 문헌 Method / 일반 프리셋 필터 및 검색
- Advanced 옵션과 Batch 비교 결과 기본 접기, 실행 버튼 고정, 읽기 쉬운 Preflight
- Queue 상태별 필터, 검색, 대기 사유 및 PC 전체 CPU/RAM 사용량 표시

## 계산 및 자원 관리
- 단일 계산, Workflow, Batch, Triplet Geometry Transfer의 공통 Auto CPU 설정; 저장 및 복원
- Windows 가용 물리 RAM 기반 MaxCore 배분, 최대 자동 동시 4개 Job, 실행 전 재검사 및 미실행 Job 재배분
- 실행 중 기존 Queue에 호환되는 Job 추가, 실행 중인 INP와 예약량 유지
- 직접 수정하거나 Import한 INP의 resource directives 유지; Manual 모드 제공
- 설정 편집 중 자동 갱신으로 경로가 덮어쓰이지 않도록 개선; 저장 실패 시 기존 설정 복원
- Triplet mapping 검토, 전하/Multiplicity, Constraints, 내부 GBW 연결 및 독립 계산 폴더 보존

## 검증 범위 및 제한
- [Windows Portable CI](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44899009033)
- [Chromium UI 회귀 테스트](https://github.com/MRYoonn/ORCA-Nova/actions/runs/44899009045)
- 모의 ORCA 프로세스 테스트: 활성 Queue 추가, RAM 부족 후 재개, 단일 계산 INP, Triplet 과학적 설정 보존, CPU 설정 저장/복원.
- 실제 ORCA 6.x/MPI 실행, 실제 메모리 피크 및 과학적 결과 **미검증**. RAM 예약량은 추정값이며 hard limit이 아닙니다. CPU/RAM 모니터는 PC 전체 사용량입니다.
- 자동 동시 실행 최대 4개, 512–16000 MB/core 등의 정책은 초기 휴리스틱입니다. 자세한 정책: [사용법 및 제한](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/V1_8_AUTO_RESOURCES.md).

## 설치
새 폴더에 ZIP 압축을 풀고 ORCA_Nova_UI.exe 실행. 기존 앱에서 Beta 채널을 선택해 업데이트할 수도 있습니다. Stable 채널은 기존 v1.7.0을 유지합니다. 계산 파일 및 개인 설정은 별도로 보관하십시오. 먼저 작은 테스트 구조로 확인하십시오.
