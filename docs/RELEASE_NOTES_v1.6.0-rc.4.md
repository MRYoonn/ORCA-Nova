# ORCA Nova v1.6.0-rc.4 — XYZ 3D 미리보기와 원자 번호

**Windows Portable Pre-release (검증용)**. 정식 안정판이 아닙니다. 기존 안정판 `v1.5.1`, RC1/2/3는 유지하고 이 RC를 안정판 자동업데이트 채널에 등록하지 않습니다.

## 추가 기능

- 기본 XYZ, 추가 XYZ, Batch 행에서 각각 **3D 미리보기**를 열 수 있습니다.
- 별도 웹 라이브러리나 인터넷 없이 회전(드래그), 확대/축소(휠), 시점 초기화, 원자 인덱스 표시 토글을 지원합니다.
- 원자 클릭 시 **ORCA 0-based atom index**, 원소, 좌표(Å), XYZ 행 번호를 표시합니다. 가능한 환경에서는 인덱스를 클립보드로 복사할 수 있습니다.
- 사용자 선택 원자를 C5/C6로 **임시 표시**합니다. XYZ 원자 순서만으로 우라실 C5/C6를 자동 추정하지 않으며, 저장된 화학적 할당도 아닙니다.
- 형식이 잘못되거나 NaN·Infinity 좌표인 XYZ는 미리보기를 거부합니다. 최대 800원자·4MB 범위에서 동작합니다.

거리와 공유결합 반경으로 그린 결합선은 **시각화용 후보**입니다. 실제 Bond Order, Tautomer, Spin Density, Radical 위치나 반응 선택성을 판단하는 결과가 아닙니다.

## 기존 기능 유지

- RC3의 기본/추가 XYZ 등록 삭제와 다시 불러오기, 입력 목록 모두 비우기
- 다중 XYZ Batch 전송·선택 삭제, 2개/4개 병렬 실행 설정, CPU/RAM 안내, Queue와 계산 OUT 결과 보존
- 전자구조 프리셋, Preflight, 결과 분석, 스핀/에너지 그래프, Portable 자동업데이트의 안정판 채널 분리

## 검증

- 개발 PR [#27](https://github.com/MRYoonn/ORCA-Nova/pull/27) 병합.
- [3D + Batch Chromium QA](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42622031984) 성공.
- [Windows Go/WebView2/Portable 빌드](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42622031695) 성공.
- [기능 및 한계 문서](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/XYZ_3D_PREVIEW.md).

## 설치/주의

1. `ORCA_Nova_Portable_Windows.zip`과 `SHA256SUMS.txt`를 내려받아 SHA-256을 대조하세요.
2. **기존 안정판/RC3과 다른 새 폴더**에 ZIP 전체를 압축 해제하고 `ORCA_Nova.exe`를 실행하세요.
3. XYZ를 불러와 `XYZ 3D 미리보기`를 누른 뒤 원자를 선택하세요. 추가 XYZ 또는 Batch 각 행에도 3D 버튼이 있습니다.
4. 실제 ORCA는 별도로 합법적으로 설치해 연결해야 합니다. 본 ZIP에 ORCA/MPI 본체가 들어 있지 않습니다.

## 정식판 전 필수 검증

- 사용자 PC의 실제 WebView2에서 팝업·마우스 드래그·휠·클립보드 조작
- 설치된 라이선스 ORCA 6.x/MPI의 실제 계산/중단/자식 프로세스 정리 및 대규모 Batch 메모리 사용
- 향후 안정판 업그레이드 시 설정 보존과 실패 복구
- 계산 프리셋·여기상태·Spin Population 등 과학적 재현성 점검

이 RC는 위의 실사용/과학 검증을 완료했다고 주장하지 않습니다.
