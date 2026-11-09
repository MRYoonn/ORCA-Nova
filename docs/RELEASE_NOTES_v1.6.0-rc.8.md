# ORCA Nova v1.6.0-rc.8 — SI 분석 및 혼합수준 Gibbs 계산

**Windows Portable Pre-release.** `v1.5.1` 안정판을 대체하지 않습니다.

## 새로운 기능

### PDF · SI Method 분석
- 사용자가 고른 **텍스트형 Supporting Information PDF(≤12MB)** 또는 SI의 **Computational/DFT Method 텍스트**를 인터넷 업로드 없이 로컬에서 분석합니다.
- **Functional, Basis Set, Dispersion, CPCM/SMD, Solvent, Software, Opt/Freq/SP/TDDFT** 후보를 검출하고 **PDF 페이지·근거 문장·사용자 제공 DOI**를 함께 표시합니다.
- 현재 편집 중인 **ORCA INP와 항목별 차이**를 표시합니다. Opt+Freq 및 Single Point의 방법이 다른 경우 **단계별 후보**를 따로 보여줍니다.
- 사용자가 명시적으로 체크·확인한 **지원되는 단일 항목만** INP에 반영하며, 단계별 후보를 선택하면 확인 후 A4(Opt+Freq)/A2(SP) 계산 모드로 전환합니다.
- 다중/모호한 Method, Gaussian 전용 Basis/Software 등은 자동 적용하지 않습니다. **근거 TXT** 다운로드가 가능합니다.
- **DOI 검증·PDF OCR·논문 전문 자동 다운로드 기능은 아직 없습니다.** 본 기능은 추출 후보를 제공하는 실험적 보조 도구입니다.

### 저수준 Opt+Freq / 고수준 SP Gibbs
- 여섯 파일(`INP, OUT, XYZ × 2`)을 받아, 동일한 분자와 계산 구조임을 확인하면 `G_composite = E_high,SP + (G_low - E_low)`를 계산합니다.
- **Charge, Multiplicity, Solvation Model·Solvent, 원자 순서 및 내부 거리(0.002 Å), SCF/Opt/Freq/정상종료, imaginary frequency 개수** 검증에 실패하면 계산을 거부합니다.
- 단위는 **절대 총 에너지 Eh**, ΔG/ΔG‡는 자동 계산하지 않습니다. 표준상태와 준조화 보정은 사용자가 별도로 확인해야 합니다.
- Gaussian OUT은 지원하지 않으며 ORCA 계산 결과를 대상으로 합니다.

### 검증
- 3개 공개 SI에서 직접 검토해 구성한 Method 텍스트 사례의 분리/모호함 처리 회귀 테스트
- 별도로 생성한 디지털 PDF 텍스트 추출, 손상 PDF 거부, 세션 인증 API 검사
- Go 테스트·Windows Portable 빌드·WebView2 EXE 실행 및 Browser Playwright SI UI 테스트 통과
- 이는 해당 논문 3편의 **전체 PDF를 자동 분석·재현했다는 뜻은 아니며**, 실제 연구실 ORCA/MPI 실행 검증도 아닙니다.

## 기존 기능 유지
- 병렬 Batch Queue, 별도 작업별 INP 및 XYZ, 취소, 재시도, 3D XYZ 뷰어
- kcal/mol 기반 상대 에너지 및 TDDFT eV 병기, 분석 CSV/TXT, Portable EXE 자동 업데이트 안전장치

## 사용
1. ZIP을 기존 버전과 다른 새 폴더에 풀고 `ORCA_Nova.exe` 실행
2. 왼쪽 **Supporting Information · PDF Method 분석 / INP 대조** 펼치기
3. SI PDF 또는 Method 텍스트를 입력한 후 근거 페이지 확인·옵션 선택
4. 또는 하단 **Opt+Freq / 고수준 SP Gibbs**에서 여섯 파일을 읽혀 검증 계산

연구실 데이터와 원본 ORCA 파일은 업로드하거나 공개하지 마세요. PDF는 로컬 앱에서 임시 분석하며 저장하지 않습니다.

[사용법 및 검증 경계](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/SI_METHOD_ANALYZER.md) · [개발 PR #40](https://github.com/MRYoonn/ORCA-Nova/pull/40).
