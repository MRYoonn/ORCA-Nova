# ORCA Nova v1.6.0-rc.3 — XYZ 목록 관리 및 병렬 Batch UX

**Windows Portable Pre-release / 검증용 개발판**입니다. 정식 안정판이 아니며 자동업데이트 `releases/latest` 채널에는 제공되지 않습니다. 기존 `v1.5.1` 안정판과 `v1.6.0-rc.1`·`rc.2` 배포물은 유지합니다.

## 이번 업데이트

- **기본 XYZ 등록 삭제**: 왼쪽 Molecule 영역에서 기존 XYZ를 해제하고 다른 파일을 다시 등록할 수 있습니다. 기본 XYZ를 삭제한 뒤 추가 XYZ가 자동으로 되살아나는 문제도 방지합니다.
- **추가 XYZ 개별 삭제**: Windows ORCA Runner 아래에 등록된 추가 XYZ의 이름과 개별 삭제 버튼을 표시합니다.
- **입력 XYZ 모두 비우기**: 확인 후 기본/추가 XYZ 목록을 한 번에 정리할 수 있습니다.
- **현재 XYZ 전체 → Batch**: 현재 불러온 기본·추가 XYZ를 아래 Multi-XYZ 계산 목록에 복사합니다. 같은 이름으로 이미 등록된 XYZ는 중복 등록하지 않습니다.
- **Batch 목록 다중 선택 및 일괄 삭제**: `선택`과 `실행`을 분리했습니다. 전체 선택·선택 해제·선택 삭제·목록 비우기를 지원합니다. 삭제는 등록 목록에만 적용되며 원본 XYZ와 기존 Job Queue, OUT 파일은 유지됩니다.
- **병렬 실행 설정 개선**: 동시 실행 수를 2개/4개로 빠르게 지정하고, 실제 INP의 `%pal nprocs`, `PALn`, `%maxcore`를 읽어 전체 CPU/RAM 예산을 안내합니다.
- 기존 Multi-XYZ Job Queue, 분자별 Charge/Multiplicity 및 INP 수정, 자원 할당, 취소·재시작·결과 비교는 유지합니다.

## 동시 실행 주의

병렬 계산 수를 2개 또는 4개로 설정하더라도 계산당 CPU/RAM 예산이 부족하면 일부 Job은 대기합니다.

예시: 실제 **16코어·64GB PC**에서 각 Job을 `nprocs 8`, `%maxcore 2500`으로 설정하면 Scheduler의 25% 여유를 포함해 Job당 약 25,000MB, 두 Job당 약 50,000MB의 RAM을 예약합니다. 전체 CPU 예산 16, RAM 예산 약 50,000MB가 필요한 셈입니다. 다른 프로그램·운영체제에 필요한 메모리를 반드시 확보하세요. 실제 ORCA 사용 메모리는 예약값보다 클 수 있습니다.

## 기존 기능 유지 / 테스트

- 공개된 실제 ORCA 6.1.1 출력 5종 Golden Parser 검증
- Windows WebView2/Go Portable 빌드
- 업데이트 EXE/UI/DLL 전체 교체 및 실패 복구
- Windows에서 실제 배포된 RC2 Portable의 실행·API·Preflight 검증
- 이번 기능의 Playwright Chromium UI 회귀 검사: XYZ 삭제·재등록, Batch 다중 선택 삭제, 2/4개 설정, 기존 Queue 보존
- 구현 및 안전성 보고서: [XYZ Manager + Batch UX](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/XYZ_MANAGER_BATCH_UX.md)

## 사용 방법

1. Windows Portable ZIP과 `SHA256SUMS.txt`를 다운로드해 체크섬을 대조합니다.
2. **기존 v1.5.1 또는 RC2 폴더와 다른 새 빈 폴더에** 압축을 해제합니다. 기존 설정·계산 결과는 먼저 백업합니다.
3. `ORCA_Nova.exe` 실행 → 기본 XYZ 불러오기 → 추가 XYZ 등록 → `XYZ 전체 → Batch`를 클릭합니다.
4. Batch 목록에서 각 XYZ의 `실행` 체크, Charge/Multiplicity, INP Method, 동시 실행 수, CPU/RAM 예산을 확인합니다.
5. 전체 Preflight를 거친 뒤 `Batch 계산 시작`을 누릅니다.

## 정식판 전환 전 남은 검증

- 사용자 Windows PC에서 합법적으로 설치된 실제 ORCA 6.x/MPI 계산·중단·자식 프로세스 정리
- 실제 WebView2 GUI 클릭 동작과 대규모 Batch 동시 실행의 메모리 사용 확인
- 사용자 PC에서 안정판 v1.5.1 → 향후 정식판 업데이트 후 UI 재실행 및 복구 테스트
- Glorius SI 전체 계산 조건과 광범위한 구조·여기상태 재현성 검증

**본 RC에는 ORCA 실행 파일 자체가 포함되지 않으며, 이 배포를 실제 ORCA/MPI 실기기 검증 완료로 주장하지 않습니다.**
