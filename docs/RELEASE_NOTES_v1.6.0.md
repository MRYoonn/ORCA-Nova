# ORCA Nova v1.6.0 — 첫 1.6 안정판

**정식 Windows Portable Release.** 기존 정식판 `v1.5.1` 이후의 핵심 기능을 하나의 안정판으로 묶었습니다. 사용자가 `Stable` 채널에서 업데이트 확인을 누르거나 최신 버전에서 자동으로 확인하면 이 릴리스를 찾을 수 있습니다. 이미 설치된 `v1.5.1`의 안정판 업데이트 검색도 이 릴리스를 인식합니다.

## 핵심 변경 사항

### ORCA 계산 GUI와 Batch
- XYZ 불러오기·삭제 및 오프라인 3D 구조 뷰어(원자 index 확인), 다중 XYZ 독립 프리셋/INP 편집
- S₀ 최적화, Triplet Opt + Spin Density, S₀→S₁/T₁ Vertical TD-DFT, S₀/S₁/T₁ 분기 워크플로와 결과 단계별 관리
- Batch 실행 CPU/RAM 예약 표시, 개별 취소·재시도·프로세스 트리 정리 및 분리된 계산 디렉터리
- 연구 요약 TXT/CSV, SCF/Opt/Frequency/Spin Population, Electronic/Gibbs/Enthalpy 총에너지의 단위 구분
- 에너지 차이와 여기 에너지 `kcal/mol` 우선 표시, eV 병기 및 원본 Eh 보존. Vertical Excitation과 Adiabatic Gap을 별도 표시

### SI 기반 Method 분석(보조 기능)
- 로컬 SI PDF 또는 붙여 넣은 Computational Method 텍스트에서 Functional/Basis/Dispersion/Solvent 후보를 **PDF 페이지와 근거 문장**으로 표시
- 현재 ORCA INP와 차이를 확인한 뒤 사용자 승인으로 지원 옵션만 적용. Opt/Freq와 High-SP 후보를 분리
- `G_composite = E_high,SP + [G_low,Opt+Freq − E_low,Opt+Freq]` 계산 및 Charge/Multiplicity/기하·용매/정상종료/허수모드 확인. 결과는 **절대 에너지 Eh**이며 반응 ΔG 자동 계산이 아님
- DOI 자동 검증·OCR·전체 논문 SI 재현은 지원하지 않음. Gaussian OUT의 자동 변환·과학적 검증은 지원하지 않음

### 자동업데이트(Stable/Beta)
- **Stable(기본):** GitHub의 검증된 정식판만 검색. 이 릴리스가 최신 정식판
- **Beta(사용자가 선택):** 시험판 `rc.N`까지 포함해 버전 번호 순으로 검색, 정식 버전과 비교
- 사용자 채널 설정을 로컬 설정에 저장하고, 다음 실행 시 복원. 릴리스 노트를 확인하고 **설치 승인 후에만** 설치
- 게시된 GitHub Portable ZIP과 SHA-256 검증 → 기존 설치 파일 백업 → 파일 교체 → 재실행. 오류 시 기존 파일로 롤백. **실행 중인 ORCA Job은 업데이트 차단**

## 설치 및 개인 데이터

- `ORCA_Nova_Portable_Windows.zip` 전체를 새 폴더에 풀고 `ORCA_Nova.exe`를 실행합니다. 기존 설치의 GUI/계산 데이터가 중요하면 업데이트 전 백업하세요.
- Portable 앱 소스/배포 ZIP은 GitHub에 있으나, **사용자의 `ORCA Nova Jobs` 계산 결과, 원본 XYZ/OUT/GBW/CUBE, 로컬 `config.json`은 GitHub에 자동 업로드되지 않습니다.**
- 자동업데이트는 배포 ZIP에 들어 있는 앱 파일에만 적용하고 원본 연구 데이터는 삭제하지 않습니다. 기본 설치 폴더에 계산 파일을 함께 두지 마세요.
- ORCA 계산 엔진/라이선스/MPI는 포함되지 않습니다. 사용 중인 ORCA 설치 경로를 별도로 지정해야 합니다.

## 테스트와 남은 검증 범위

- Windows Go tests/vet, Native WebView2 host/Portable 패키징 및 앱 실행·인증 API 테스트, Browser QA, SI 분석/보조 계산식 테스트 및 공개 ORCA 6.1.1 **OUT 파서** 회귀 테스트를 통과한 코드입니다.
- 이전 공개 ZIP에서 파일 교체·실패 시 롤백을 수행하는 Windows 테스트를 확인했습니다.
- **실제 연구실 PC에서 ORCA 6.x/MPI 계산 성공, 여러 실제 ORCA 작업 동시 실행 및 사용자 업데이트 승인/재실행의 물리적 기기 검증은 별도 확인이 필요합니다.** 이런 과학적 검증이 끝났다고 주장하지 않습니다.

[SI 분석 설명](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/SI_METHOD_ANALYZER.md) · [Windows ORCA6 실사용 검증](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/REAL_WINDOWS_ORCA6_ACCEPTANCE.md)
