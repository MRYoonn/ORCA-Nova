# ORCA Nova v1.6.0-rc.6 — kcal/mol 기본 결과 표시

Windows Portable **Pre-release** (시험판). **정식 안정판 v1.5.1 및 기존 rc.5 릴리스는 그대로 유지**합니다. 안정판 자동업데이트로 강제 전환하지 않습니다.

## 이번 변경 사항

- **Relative Energy / PES / Optimization 그래프** 기본 단위를 `kcal/mol`로 변경. 단위 메뉴에서 `Eh`, `kJ/mol`, `eV`를 계속 선택할 수 있습니다.
- 사용자가 출처 OUT와 계산 조건을 지정·확인한 **S₀↔T₁ Adiabatic Gap 및 Reaction Profile**에서 `ΔE`, `ΔG`, `ΔG‡` 수치를 **kcal/mol 우선**, kJ/mol 병기로 표시하며 프로파일 그래프 Y축도 kcal/mol로 표기합니다.
- **TD-DFT S₁/T₁ Vertical Excitation**을 kcal/mol 우선, eV 병기. 각 Root 표에는 파장(nm)과 oscillator strength가 유지됩니다. Excitation 그래프의 단위도 선택할 수 있습니다.
- **Batch 계산별 비교표**에서 Vertical S₁/T₁을 `kcal/mol (eV)`로 표시합니다.
- **절대 Electronic/Gibbs/Enthalpy Energy**는 ORCA의 `Eh` 원본을 유지하고, ZPE는 총에너지가 아닌 보정항으로 구분합니다. 백엔드의 과학적 검증 로직 및 원본 kJ/mol 수치는 바꾸지 않습니다.
- 1 eV = 23.060548867 kcal/mol; 1 kcal = 4.184 kJ. 원본 절대 에너지를 임의로 반응 에너지로 변환하지 않고, Vertical/Adiabatic T₁을 혼동하지 않습니다.

## rc.5 기능 그대로 유지

- XYZ 오프라인 3D 미리보기, 0-based 원자 index 확인 및 임시 C5/C6 수동 지정
- 여러 XYZ 동시 Batch Queue, 분자별 독립 계산 프리셋, INP 편집/불러오기, 선택·전체 삭제
- 수렴 상태와 Electronic/Gibbs/Enthalpy/ZPE, Spin Population, ⟨S²⟩, TD-DFT Root의 읽기 쉬운 결과 요약
- 원본 OUT·고급 분석 JSON, 오류/주의사항 및 CPU/RAM 스케줄링

## 검증 및 설치

- [PR #33](https://github.com/MRYoonn/ORCA-Nova/pull/33) 반영. Chromium Browser QA와 Windows Portable Build (Go tests/vet, WebView2 GUI, ZIP 패키징) 성공.
- [kcal/mol 표시 및 연구 해석 기준](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/KCAL_PRIMARY_RESULTS.md)
- 기존 실행 폴더를 덮어쓰지 않고 새 폴더에 ZIP 전체를 압축 해제하세요. 계산 결과와 설정을 백업한 뒤 `ORCA_Nova.exe`를 실행하세요.
- 실제 연구실 PC의 ORCA/MPI 동시 계산, 취소/재시작, 자동업데이트 재실행 검증은 별도로 필요합니다. ORCA 실행 파일은 동봉하지 않습니다. 실제 계산을 자동으로 완료하거나 결과의 과학적 타당성을 보증하지 않습니다.
