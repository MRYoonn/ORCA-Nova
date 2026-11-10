# ORCA Nova v1.6.1 — 프리셋 가독성·접기 UI 업데이트

**정식 Windows Portable Stable 패치**. `v1.6.0`의 계산 기능과 자동업데이트 기능을 유지하면서 UI를 개선했습니다.

## 바뀐 화면
- **⚡ 빠른 계산 프리셋:** `문헌 Method → 추천 → 프리셋` 순서로 그룹을 분리합니다. 그룹 안에서는 각 프리셋이 **왼쪽에서 오른쪽, 다음 행** 순서로 2열 배치됩니다.
- **제목 읽기 편하게:** 계산 설정 패널을 넓혀 불필요하게 2줄로 끊기는 프리셋 제목을 줄였습니다. 제목과 Method 요약을 분리하고 그룹별 라벨/개수를 표시합니다.
- **Calculation Library 접기:** 프로그램 첫 화면은 기본적으로 **접힌 상태**입니다. ☰ 버튼으로 언제든지 펼치거나 닫을 수 있습니다.
- **라이브러리 그룹별 접기:** 펼친 라이브러리에서도 W/A/B/C/D/E 분류가 각각 접힌 상태로 시작합니다. 개별 그룹을 열고 닫을 수 있고 검색 시 일치 그룹만 임시로 자동 펼칩니다.
- 기존 프리셋의 Method, Basis, Dispersion, Solvent, 계산 모드와 INP 생성 동작은 변경하지 않았습니다.

## 테스트
- Chromium 실제 화면 회귀 테스트에서 1600px 기준 프리셋 제목 한 줄, 2열 왼쪽→오른쪽 카드 순서, 기본 Library 접기/펼치기·그룹 선택·검색·모바일 폭을 검증했습니다.
- Batch/다중 XYZ/3D XYZ/SI PDF 분석/Stable·Beta 업데이트의 기존 브라우저 테스트도 통과했습니다.
- Windows Go 테스트·`go vet`·WebView2 Native Host·Portable ZIP 패키징·EXE와 인증된 API 실행 Smoke Test 통과.
- 기존 ORCA 6.x/MPI **연구실 실기기 계산 검증은 여전히 별도 작업**입니다.

## 설치 및 백업
- 기존 앱에서 **Stable → 업데이트 확인 → v1.6.1 설치 승인**으로 업데이트할 수 있습니다. 새 설치는 릴리스의 Portable ZIP을 압축 해제해 사용하세요.
- 기존 `v1.6.0` 릴리스는 GitHub에 그대로 남겨 두며, ORCA 계산 파일과 `config.json`은 GitHub에 자동 백업되지 않습니다. 중요 데이터는 별도 보관하세요.

개발/리뷰: [PR #47](https://github.com/MRYoonn/ORCA-Nova/pull/47). UI 회귀: [GitHub Actions](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42826871603).
