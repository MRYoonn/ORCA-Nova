# ORCA Nova v1.7.0 — Triplet Geometry Transfer + Spin

**Windows Portable 정식 배포 (Stable 채널).** v1.7.0-rc.1의 기능을 Stable 업데이트 채널로 제공합니다. 정식 배포는 Windows 앱·패키징의 검증 수준을 뜻하며, **실제 ORCA 6.x/MPI Triplet 계산의 과학적 검증을 완료했다는 의미가 아닙니다.**

## 새 기능
- **Triplet Geometry Transfer + Spin:** 최적화된 Reference Triplet XYZ의 C2–N3–C4–C5–C6–N1 고리 구조를 다른 S-alkyl thiouracil Target에 매핑·이식합니다. Target의 측쇄·화학식·원자 순서를 보존하고 잘못된 매핑·전하·충돌을 경고합니다.
- STEP 0 좌표 이식 → STEP 1 고리 Constrained Opt → STEP 2 Unconstrained UKS Triplet Opt (Frequency 선택) → STEP 3 GBW를 사용하는 Spin Density Single Point.
- C2/C5/C6 Spin Population(Mulliken/Löwdin/Hirshfeld), 주요 결합 길이, ⟨S²⟩, 에너지, Cube/GBW 여부를 확인하고 CSV/JSON으로 출력합니다.
- 기존 W2/W5 프리셋, Batch Scheduler, 자원 제한, 단계별 실패 분석을 유지합니다.

## 테스트 범위 및 과학적 주의
- [Windows Portable 빌드/Go tests/EXE·API smoke test](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43059990398), [공개 RC ZIP 무결성 감사](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43059388469), [실제 Windows EXE 업데이트·롤백 검증](https://github.com/MRYoonn/ORCA-Nova/actions/runs/43059555059)이 성공했습니다. 정식판도 동일 테스트를 새 버전 번호로 다시 실행하여 빌드합니다.
- **실제 ORCA 6.x/MPI를 사용한 해당 기질의 Geometry Optimization, Frequency, Spin Density, SOMO/Cube, 상태 정체성 및 수치 타당성은 미검증**입니다. 사용 전 생성된 INP의 Charge/Multiplicity, 매핑, 수렴 기준을 확인하고, 계산 후 ⟨S²⟩·진동수·전자 상태를 검토하세요.
- Frequency를 생략한 구조는 최소점으로 검증되지 않습니다. 다른 기질의 절대 에너지를 직접 비교하지 마세요.

## 자동업데이트·백업·복구
- 기존 **v1.6.1 Stable**에서 앱의 **Stable → 업데이트 확인 → 설치 승인**으로 v1.7.0을 설치할 수 있습니다. 기존 [v1.6.1 정식판](https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.6.1)과 [v1.7.0-rc.1 시험판](https://github.com/MRYoonn/ORCA-Nova/releases/tag/v1.7.0-rc.1)은 GitHub에서 그대로 보존합니다.
- 자동업데이트 시 기존 프로세스 종료 후 **교체 대상 프로그램 파일(EXE/DLL/설정 구성 파일 등)을 먼저 복사**하여 설치 폴더의 `.ORCA_Nova_backup_yyyyMMddHHmmssfff`에 남깁니다. 파일 교체 또는 재실행이 실패하면 이전 파일을 자동 복원하고 새로 추가된 파일을 제거합니다.
- 백업 폴더는 업데이트 성공 후에도 삭제하지 않습니다. 기존 개인 파일과 계산 파일은 교체하지 않지만 **GitHub나 클라우드로 자동 백업되는 것은 아닙니다.** 중요 XYZ/INP/OUT/GBW/CUBE 및 `%APPDATA%\ORCA Nova\config.json`은 별도로 보관하세요.
- 새 설치: 이 릴리스의 `ORCA_Nova_Portable_Windows.zip` 전체 압축 해제 후 `ORCA_Nova.exe` 또는 UI 호스트를 실행하세요. ORCA 프로그램 자체는 포함하지 않습니다.

자세한 사용법: [Triplet Geometry Transfer](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/TRIPLET_GEOMETRY_TRANSFER.md).
