# ORCA Nova v1.6.0-rc.1 — 개발·검증용

이 버전은 Pre-release이며 정식 자동업데이트 대상이 아닙니다. 실제 ORCA 계산 정확성, Windows ORCA/MPI 실행·중단 및 실기기 자동업데이트 검증이 완료되지 않았습니다.

## 주요 변경
- Multi-XYZ/INP Batch, CPU/RAM 예약 기반 병렬 Queue, 작업 취소·복구·실패 단계 재시작
- OUT 분석, TXT/JSON/CSV 보고서와 PES·SCF·Optimization·TD-DFT·Frequency·Spin 시각화
- S₀/T₁ 및 Reaction Profile, Cross-Job 비교의 조건·조성 검사 강화
- 기존 다크 테마와 한국어 도움말, Portable 구조 유지
- ZIP 경로 검증과 업데이트 회귀 검사 강화

## 설치·검토
`ORCA_Nova_Portable_Windows.zip`과 `SHA256SUMS.txt`를 다운로드하고 SHA-256을 확인하세요. ZIP 전체를 새 폴더에 풀어 기존 설치와 분리해 검토하세요. Windows 10/11, .NET Framework 4.8 및 WebView2 Runtime이 필요합니다. ORCA는 포함되지 않으며 사용자가 합법적으로 설치한 ORCA 6.x가 별도로 필요합니다. 기존 설정과 계산 결과는 먼저 백업하세요.

## 알려진 제한
실제 ORCA Golden 데이터 검증, Windows ORCA/MPI 실계산·취소, v1.5.1에서의 실기기 업데이트·재실행은 미완료입니다. 업데이트 실패 시 전체 UI DLL을 포함한 트랜잭션 복구도 완전하지 않습니다. 본 RC는 기존 안정판 업데이트 알림으로 제공하지 않습니다. Glorius 원문 SI·계산 데이터의 연결 및 세부 조건은 접근 가능한 근거 이상으로 확정하지 않습니다. 합성 Fixture와 Mock 테스트를 과학적 검증으로 취급하지 마세요.
