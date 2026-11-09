# ORCA Nova — 실제 Windows/ORCA 6.x 정식판 승인 체크리스트

이 문서는 `v1.6.0-rc.2`의 **실제 연구실 PC 검증**을 위한 것이다. GitHub CI의 Mock ORCA·PowerShell 검증과 실제 사용자 PC 동작은 구별한다.

## A. 실행 환경과 기존 파일 보호

- [ ] 실험용 Windows 10/11 PC에 .NET Framework 4.8과 WebView2 Runtime 확인
- [ ] 실제 ORCA 6.x의 합법적인 설치 경로와 `orca.exe` 실행·버전 확인
- [ ] 기존 안정판 `v1.5.1`의 작업 폴더, 설정 및 최근 계산 파일을 **외부 폴더에 백업**
- [ ] RC.2 ZIP 전체를 새 빈 폴더에 압축 해제 (기존 안정판 **덮어쓰기 금지**)
- [ ] RC.2에서 `ORCA_Nova.exe` 실행 후 실제 네이티브 GUI와 작업 표시줄 아이콘 확인

## B. 최소 실제 계산 — RC.2에서 실행

실험용 XYZ만 사용하고 미공개 분자/논문 계산물을 GitHub에 업로드하지 않는다.

- [ ] H₂O Single Point, ORCA 정상 종료 문구와 OUT 최종 에너지 확인
- [ ] H₂O Opt + Freq: Geometry 수렴·허수 모드 0개·E/Gibbs 및 출력 표 비교
- [ ] 소형 Doublet 라디칼: Charge/Multiplicity·⟨S²⟩·Spin Population 확인
- [ ] 소형 Triplet: ⟨S²⟩, 원자별 Spin Population, density/cube의 유무를 분리 확인
- [ ] TD-DFT: S/T Root와 Vertical Excitation Energy·oscillator strength 확인
- [ ] 작은 Relaxed PES Scan: 실제 Å/° 좌표, 수렴 상태, 에너지 그래프, CSV/SVG/PNG 확인
- [ ] 서로 다른 XYZ 2개를 CPU/RAM 제한 내 병렬 Job 실행, OUT/GBW 폴더가 섞이지 않는지 확인
- [ ] 둘 중 하나만 취소하여 ORCA/MPI 자식 프로세스가 종료되고 다른 Job은 계속되는지 작업 관리자에서 확인
- [ ] 강제 종료 이후 Queue 복구·재시도, 이전 정상 결과 보존 확인

## C. 업데이트·복구 실기기

- [ ] 안정판 `v1.5.1` 별도 테스트 복사본에서 업데이트 버전 확인 (RC는 안정판 업데이트에 제시되지 않아야 함)
- [ ] 릴리스 검증용 격리된 테스트 폴더에서 안전한 Mock/staged ZIP 교체 및 WebView2 UI 재실행
- [ ] 실패를 의도적으로 일으킨 뒤 EXE, UI EXE, DLL, 하위 DLL 파일 및 사용자 설정을 모두 대조해 원상복구 확인
- [ ] 정상 교체 후 재실행, 설정·작업 폴더 보존, 업데이트 로그 기록 확인
- [ ] 실제 사용자 PC의 완료되지 않은 계산이 있으면 업데이트 차단 확인

안정판 자동업데이트는 현재 `v1.5.1`을 기준으로 동작한다. `v1.6.0-rc.2`를 정식 채널로 승격하지 않았으므로, **기존 안정판 앱에서 RC.2로 자동 전환되기를 기대하거나 강제로 교체하지 않는다.**

## D. 보고서

검증 컴퓨터에서는 환경 정보(Windows 버전, ORCA 버전, WebView2 및 MPI 버전), Job 상태, 실제 OUT과 UI 표시값 비교, 종료된 PID, 오류 로그, ZIP SHA-256을 기록한다.

민감한 분자 구조, 미공개 데이터, 개인 계정 경로·환경변수·인증정보는 공개 GitHub에 업로드하지 않는다. 실패한 단계의 비민감 오류 메시지만 공유한다.

정식 `v1.6.0` 발행 기준: 해당 실기기 테스트 통과, 알려진 중대한 회귀 없음, Golden 데이터 범위·제한 명시, 완료된 릴리스 승인 기록.

**결과 기록 전에는 정식판 검증 완료라고 판단하지 않는다.**
