# ORCA Nova v1.7.0-rc.1 — Triplet Geometry Transfer + Spin

**Windows Portable 공개 시험판 (Beta/Release Candidate). 실제 ORCA 6.x 계산으로 최종 화학적 검증을 거치지 않았습니다.** 기존 Stable `v1.6.1`은 그대로 유지됩니다. 이 버전은 연구 분석용 시험판이며 검증되지 않은 Triplet 에너지·Spin 결과를 확정된 과학적 사실로 취급하지 마세요.

## 새 기능
- **Triplet Geometry Transfer + Spin** 프리셋 추가. 기존 `W2`와 `W5`는 변경하지 않았습니다.
- 이미 최적화된 Reference Triplet XYZ와 다른 유도체의 Target XYZ(복수 선택)를 불러와 **C2–N3–C4–C5–C6–N1**의 화학적 원자 매핑을 확인합니다. 고리 S/O 및 고리 H도 선택할 수 있습니다.
- Reference 고리 구조를 Target에 정렬해 이식하고, Target 측쇄·화학식·원자 순서는 보존합니다. 매핑 모호성, protonation/전하 불일치, 충돌·비정상 결합 길이를 경고합니다.
- **STEP 0**: `transferred_initial.xyz` 좌표 이식 → **STEP 1**: 고리 고정 Constrained Opt → **STEP 2**: 제약 없는 UKS Triplet Opt (Frequency 선택) → **STEP 3**: STEP 2 GBW를 사용하는 Spin Density Single Point.
- 기존 Batch Scheduler로 Target별 독립 디렉터리를 생성하고 CPU/RAM 예산, 동시 처리, 취소·단계별 실패 분석 기능을 이용합니다.
- C2/C5/C6의 Mulliken/Löwdin/Hirshfeld Spin Population, 주요 결합 길이, 최종 `⟨S²⟩`, 계산 에너지 및 Cube/GBW 생성 여부를 표시하고 CSV/JSON으로 출력합니다.
- 기존 W2와 Transfer를 **같은 기질·같은 Method·같은 Charge/Multiplicity 및 수렴 조건일 때만** 상대 에너지로 비교합니다. 서로 다른 유도체의 절대 전자 에너지로 안정성을 판단하지 않습니다.
- Frequency를 건너뛰면 **최소점 미검증**으로 표시하고, UKS 계산 성공만으로 **최저 T₁ 상태의 정체성**을 주장하지 않습니다.

## 완료한 소프트웨어 테스트
- [Windows Portable 빌드, Go Mock ORCA Batch 및 로컬 API 검사](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43056145539)
- [브라우저 UI 및 기존 XYZ/Batch/W2/W5/SI/업데이트 회귀 검증](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43056145593)
- 시험용 ORCA 수용시험 PowerShell과 빌드 오류 분석 파일 제공: [사용 설명서](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/TRIPLET_GEOMETRY_TRANSFER.md)

**테스트의 ORCA 계산은 가상(Mock) 실행입니다.** Windows EXE 시작·버전 표시·API 및 파일 교체 가능성을 시험했지만 **실제 S-alkyl thiouracil에 대한 ORCA 6.x/MPI의 3단계 Opt·Spin Density·SOMO·Cube·진동수·전자배치의 과학적 타당성을 검증하지 않았습니다.**

## 설치 및 업데이트
1. 기존 정식판 설치 폴더 및 계산 파일을 먼저 보존하세요. GitHub는 사용자 PC의 XYZ/OUT/GBW/CUBE 또는 개인 설정을 자동 백업하지 않습니다.
2. 새 폴더에 이 페이지의 `ORCA_Nova_Portable_Windows.zip`을 압축 해제해 `ORCA_Nova_UI.exe`를 실행하거나, 기존 앱의 업데이트 설정에서 **Beta 채널**을 선택한 뒤 업데이트를 확인하고 승인하세요.
3. 첫 계산은 원본이 아닌 복제된 작은 구조로 시험하고, `Reference`와 `Target`의 N1 protonation, Charge, atom mapping 및 출력 결과를 직접 확인하세요.
4. 문제가 발생하면 기존 [v1.6.1 Stable](https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.1)을 사용하세요. 자동 업데이트는 Stable 채널 사용자를 이 시험판으로 강제 변경하지 않습니다.

**정식 Stable 승격은 실제 ORCA 6.x 결과 검증 전까지 보류합니다.**
