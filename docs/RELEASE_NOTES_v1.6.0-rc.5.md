# ORCA Nova v1.6.0-rc.5 — Batch 계산별 프리셋 및 결과 요약 개편

**Windows Portable Pre-release / 실사용 확인용 개발판**. 정식 안정판이 아니며, 안정판 자동업데이트 채널에는 등록하지 않습니다. 기존 `v1.5.1` 안정판 및 rc.1~rc.4는 변경하지 않습니다.

## 이번 주요 업데이트

### 1. XYZ마다 다른 계산 지정

- Batch 표에서 각 분자에 **계산 프리셋(A/B/C/D/E/W)**을 독립적으로 설정할 수 있습니다. 예: monomer에 `W2 S₀ Opt → T₁ Opt + Spin Density`, dimer에 `A4 Opt + Freq`.
- 해당 행의 프리셋만 바꾸며 다른 분자의 INP, 왼쪽 기본 프리셋, 이미 제출한 Queue와 OUT는 변경하지 않습니다.
- 직접 수정하거나 외부에서 불러온 INP를 덮어쓸 때는 확인 대화상자를 표시합니다.
- 기존의 `현재 프리셋을 모두 적용`은 요청한 경우에만 전체 Batch를 다시 생성하는 명시적 동작을 유지합니다.

### 2. 사람이 읽는 계산 결과 요약

기존 `결과 요약` 탭의 원시 JSON 표시를 계층형 연구용 보고 화면으로 대체했습니다.

1. **계산 상태**: ORCA 정상 종료, SCF convergence, Geometry Opt 수렴을 구분 표시.
2. **총 에너지 3종과 보정항 1종**: Electronic Energy, Gibbs Free Energy, Enthalpy, Zero-Point Energy (ZPE correction)를 구분해 표시(단위 Eh).
3. **Spin**: 값이 있으면 ⟨S²⟩과 해당 Multiplicity의 이상적 S(S+1)을 나란히 표시.
4. **TD-DFT Vertical Excitation**: ROOT, eV, nm, oscillator strength로 표 구성.
5. **Atomic Spin Population**: 원소, **ORCA 0-based 원자 번호**, 부호가 있는 Spin 값, 분석법을 명시.
6. **세부 정보**: 경고, 전체 분석 JSON, 원본 OUT 탭을 유지하며 연구노트용 요약 복사 버튼을 추가.

계산되지 않은 항목은 **Not calculated**로 표시하며 가상의 수치를 보충하지 않습니다. Vertical Excitation은 해당 구조에서의 값이지 자동으로 Adiabatic T₁ Gap이 아닙니다. 절대 에너지를 임의의 반응 에너지로 바꾸지 않으며, C5/C6 위치나 반응 선택성을 Spin 수치만으로 확정하지 않습니다.

## 유지 기능

- rc.4의 오프라인 3D XYZ 뷰어, ORCA 원자 인덱스 확인 및 임시 C5/C6 지정
- 기본/추가 XYZ 등록 해제, Batch 선택/전체 삭제, 동시 실행 2/4 설정, CPU/RAM 예산 안내
- Preflight, Batch Queue, 개별 INP 편집, 원본 결과 분석 및 파일 보존

## 테스트

- 개발 PR [#30](https://github.com/MRYoonn/ORCA-Nova/pull/30) 병합 및 새 Chromium 회귀 검사 통과.
- Windows Go 테스트/vet, WebView2 native UI 빌드와 Windows Portable ZIP 패키징 통과.
- 구현·과학적 해석 제한: [Batch Presets and Readable Results](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/BATCH_PRESETS_READABLE_RESULTS.md).

## 설치 및 검증상 한계

기존 rc.4/안정판과 **다른 새 폴더**에 ZIP 전체를 압축 해제한 후 `ORCA_Nova.exe`를 실행하세요. 기존 저장된 계산 결과와 설정은 반드시 백업하세요. 실제 ORCA 6.x/MPI는 사용자가 별도로 합법적으로 설치해 연결해야 합니다. 본 배포물에는 ORCA 실행 파일이 포함되지 않습니다.

이 RC는 Windows 연구실 PC에서의 실제 ORCA/MPI 동시 실행, 작업 중단 및 프로세스 정리, 실제 자동업데이트/재실행, 포괄적 계산화학 검증을 완료했다고 주장하지 않습니다. 안정판 승격 전에 위 실사용 검증이 필요합니다.
