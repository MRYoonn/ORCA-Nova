# ORCA Nova · Stable / Beta 자동업데이트 채널

## 선택과 지속성

- 앱 최초 실행 시 `Stable`이 기본입니다. GitHub 공식 `releases/latest`를 조회하며 `prerelease=true`인 시험판은 무시합니다.
- 사용자가 `Beta`를 선택하면 최신 GitHub Releases 목록에서 `vMAJOR.MINOR.PATCH-rc.N`을 포함해 버전 순으로 가장 높은 사용 가능한 릴리스를 찾습니다. rc.9→rc.10 정렬이 숫자로 처리되고 같은 버전의 정식판이 rc보다 우선합니다.
- 선택 채널은 `%APPDATA%\ORCA Nova\config.json` (OS별 `UserConfigDir` 경로)에 저장됩니다. 사용자가 ORCA 실행 경로/작업 폴더를 변경해도 채널이 지워지지 않습니다.
- 채널 변경 시 이전 조회 결과 및 설치 대상은 무효화합니다. 검색 중 채널을 변경할 수 없고, 설치 중 변경을 차단합니다.
- 자동으로 실행할 때는 **릴리스 조회만** 수행하며 사용자의 확인 없이 ZIP을 설치하지 않습니다.

## 안전장치

- 항상 GitHub 발행 ZIP과 `SHA256SUMS.txt`가 모두 있는 정상 릴리스만 후보로 제시합니다.
- 전송 URL/리디렉션에 신뢰 호스트 제한, HTTPS, 다운로드 크기 제한과 SHA-256 검증을 적용합니다.
- 업데이트 대상은 허용된 Portable 바이너리/런타임만 포함합니다. 경로 탈출/예기치 못한 스크립트/중복 파일/사용자 설정 파일은 거부합니다.
- 실행 중인 단일 Job 및 Batch Queue에 실행·대기 작업이 있으면 설치를 차단합니다.
- 파일별 설치 이전 백업, 교체 실패 시 롤백, 성공/실패 로그를 유지합니다.
- **GitHub는 소스코드·릴리스 ZIP의 백업일 뿐, 사용자의 연구 데이터·설정 파일 백업이 아닙니다.** 수동으로 개인 계산 자료를 백업하는 것을 권장합니다.

## 검증

- `update_channels_test.go`: 기본 Stable/Beta 분리, rc 버전순 비교, 누락 파일/초기화, 세션 보호 및 채널 설정 저장
- `tests/update-channels-ui.cjs`: Stable/Beta 선택, 릴리스 노트, 사용자 설치 승인
- `update_windows_transaction_test.go`: 설치파일 교체/실패 롤백, 설정 보존
- `update_published_releases_test.go`: 기존 공개 릴리스에서 새로운 ZIP으로 파일 교체 및 복구
- Windows Portable workflow: 빌드 후 EXE 직접 실행/로컬 API/버전 확인

## 정식판 릴리스 게이트

- `main`에 승인된 릴리스 요청 PR을 병합해야 하며 `release_type=stable`, `stable_release_blocked=false`, `prerelease=false` 명시가 필요합니다.
- 새 버전으로 Go tests/vet, Windows GUI/Portable 패키징, **실제 배포 EXE/API/버전 smoke test**가 통과한 후에만 GitHub Release를 게시합니다.
- 안정판 배포 후 공개 ZIP SHA-256·CRC·버전 및 GitHub latest 안정판을 별도로 검증합니다.
