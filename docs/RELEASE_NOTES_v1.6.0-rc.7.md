# ORCA Nova v1.6.0-rc.7 — 연구 결과 보고서·Batch 안정성 검증·UI 개선

**Windows Portable Pre-release / 실사용 검증용 개발판**입니다. 기존 안정판 `v1.5.1` 및 `rc.6`를 변경하지 않습니다. 연구실 ORCA/MPI 검증 전 정식 안정판으로 승격하지 않습니다.

## 이번 업데이트

### 1. Batch 결과 보고서 확장
- **↓ 전체 CSV**에 분자별·Stage별 실제 계산 상태, ORCA 정상 종료, SCF 수렴, Opt 수렴, Imaginary Frequency 수, ⟨S²⟩, Electronic/Gibbs/Enthalpy (총 에너지, Eh), ZPE 보정항 (Eh), S₁/T₁ Vertical (kcal/mol과 eV), 온도와 경고를 추가합니다.
- **↓ 전체 TXT** 버튼을 추가하여 분자·단계별 Method, Geometry, 에너지와 Spin Population(ORCA 0-based 인덱스, 부호 보존)을 읽기 쉬운 텍스트로 저장합니다.
- 미수렴·누락 OUT의 숫자를 정상 결과처럼 표시하지 않습니다. 기존 `analysis.txt` 및 `results.json`을 보존하고, 서로 다른 분자의 절대 에너지로 반응 ΔG를 자동 생성하지 않습니다.
- 단계별 해석 TXT는 **ΔE/ΔG·Vertical Excitation을 kcal/mol 중심으로, 절대 에너지를 Eh**로 구분합니다. Vertical은 eV 병기입니다.

### 2. Batch·UI
- Queue 상태 줄에 **실행 중 Job 수 / 예약된 CPU·RAM / 전체 예산**을 표시합니다. 메모리는 `%maxcore × nprocs`에 안전 여유를 반영한 **예약 추정치**이지 실제 RAM 사용량이 아닙니다.
- Functional/Basis 선택창의 글자 크기·설명 간격 개선. 추천/프리셋/문헌 Method 배지 의미를 명확히 구별하고, 한국어 도움말을 유지합니다.
- 등록된 XYZ 삭제·분자별 계산 지정·기존 Workflow 재시작·3D XYZ Viewer 등 rc.6 기능은 그대로 유지합니다.

### 3. 테스트/진단 자동화
- **기존 Batch 테스트 강화:** 실제 Windows/Unix 프로세스에서 Mock 4-worker CPU/RAM 예약 검사, 선택 Job만 중단한 후 다른 Job 계속 실행, 생성물 폴더 격리.
- **CSV/TXT 정확성:** 실제 OUT에서 추출된 값만 표기, 결측·실패 시 값 차단, 미확인 ΔE/ΔG 추정 금지, 세션 인증된 다운로드.
- Windows Go tests/vet, WebView2 호스트와 Portable ZIP 패키징, **생성된 ZIP의 Windows EXE 실행·인증 API 테스트** 성공.
- 공개 ORCA 6.1.1 출력 파일의 파서 골든 테스트 통과 (ORCA 실행 자체는 아님).
- 공개 `v1.5.1`/ `rc.6` ZIP을 직접 다운로드하여 무결성·EXE 실행·분리된 파일 업데이트/롤백 테스트 성공.
- 실제 연구실 Windows ORCA 6.x 출력의 OUT/XYZ/GBW/CUBE 확인 도구 `tests/windows-orca6-smoke-check.ps1`와 실행 절차 `docs/REAL_WINDOWS_ORCA6_ACCEPTANCE.md` 제공. 해당 도구는 파일을 **로컬 검사만** 하고 공개 저장소에 전송하지 않습니다.

## 남아 있는 실사용 검증 (안정판 승격 전에 필요)
- 연구실 Windows PC에서 ORCA 6.x/MPI를 연결해 실제 `S₀ Opt + Freq → T₁ Opt + Spin Density` 계산 완료·출력 파일 검증
- 2~4개 실제 ORCA/MPI 작업의 분리, 개별 취소, 프로세스 정리 및 파일 보존
- 작업 표시줄 범고래 아이콘과 사용자 승인형 업데이트 UX, 재실행·롤백의 실제 PC 확인

**설치:** 기존 버전의 폴더를 덮어쓰지 말고 새 폴더에 Portable ZIP을 전체 압축 해제하세요. 계산 원본 파일과 설정을 백업하세요. ORCA/MPI 실행 파일은 포함되지 않습니다.

개발 PR: [#36](https://github.com/MRYoonn/ORCA-Nova/pull/36). 연구실 PC 점검 절차: [Windows ORCA 6.x Acceptance](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/REAL_WINDOWS_ORCA6_ACCEPTANCE.md).
