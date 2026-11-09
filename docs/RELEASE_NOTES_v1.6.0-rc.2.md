# ORCA Nova v1.6.0-rc.2 — Windows Portable 검증용

**Pre-release (정식판 아님).** 기존 `v1.5.1` 안정판 자동업데이트에는 제공하지 않습니다. `v1.6.0-rc.1`도 그대로 보존됩니다.

## 이번 RC에서 추가된 개선

- **실제 ORCA 6.1.1 OUT 5종 Golden Test**: H₂O Single Point 및 Opt+Freq의 Electronic/Gibbs/Enthalpy/ZPE/Frequency, 에탄 Relaxed PES Scan 5단계, 벤젠 TD-DFT Singlet Root, Triplet O₂ Spin Population 및 ⟨S²⟩ 대조. 공개 원본 데이터를 CI에서 임시 다운로드하고 검사 후 제거하므로, 원본 ORCA 출력이나 ORCA 실행 파일은 배포 ZIP에 포함되지 않습니다.
- **PES Scan 좌표 정확도**: 실제 ORCA Scan Step 헤더의 Bond/Angle/Dihedral 좌표를 판독하고 그래프에 맞는 단위를 표시합니다.
- **자동업데이트 전체 파일 복구 강화**: 실패 시 기존 Backend EXE뿐 아니라 WebView2 UI EXE/DLL과 하위 폴더 파일도 함께 원상 복구합니다. 새로 설치한 파일은 복구 중 제거하며 기존 개인 설정을 건드리지 않습니다.
- **Windows PowerShell 5.1 호환성**: 한글이 포함된 업데이트 스크립트의 UTF-8 BOM 저장 오류를 수정했습니다.
- **실제 Windows 테스트**: GitHub Windows Runner에서 업데이트 성공 및 강제 실행 실패 시 EXE/UI/DLL 복구 테스트를 통과했습니다.
- 이전 RC의 Multi-XYZ Batch, 병렬 Queue, 계산 상태 재개, 결과 시각화, S₀/T₁·Reaction Profile과 Cross-Job 분석을 유지합니다.

## 설치 및 수동 검증

1. `ORCA_Nova_Portable_Windows.zip`과 `SHA256SUMS.txt`를 받아 SHA-256을 대조합니다.
2. **기존 안정판 폴더와 다른 새 폴더에 압축을 모두 풉니다.** 기존 작업 및 설정은 먼저 백업합니다.
3. Windows 10/11에서 .NET Framework 4.8, WebView2 Runtime 설치를 확인합니다.
4. 합법적으로 설치한 ORCA 6.x 경로를 별도로 설정합니다. ORCA 자체는 이 ZIP에 포함되지 않습니다.

## 남아 있는 정식판 차단 조건

- 실제 Windows PC에 설치한 ORCA 6.x와 MPI로 계산·중단·자식 프로세스 정리 검증 **미실시**
- 실제 `v1.5.1`을 실행한 사용자 환경에서 자동업데이트·새 UI 재실행·복구까지 검증 **미실시** (Windows CI의 격리된 PowerShell 테스트와 다름)
- 실제 ORCA 출력 5종 외 복잡한 여기상태·Scan 계산 등 광범위한 과학적 정확성 검증 필요
- Glorius SI의 전체 계산 설정·조건 재현성은 추가 검증 필요

**이 RC의 테스트 통과는 위 미완료 작업의 성공을 의미하지 않습니다. 정식 안정판 v1.6.0은 아직 배포하지 않습니다.**

## 검증 자료
- [실제 ORCA Golden CI](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42573626339)
- [Windows 업데이트 복구 CI](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42574801764)
- [통합 `main` 빌드 CI](https://github.com/MRYoonn/ORCA-Nova/actions/runs/42575117822)
- [실제 ORCA Golden 검증 보고서](https://github.com/MRYoonn/ORCA-Nova/blob/main/docs/REAL_ORCA_GOLDEN_VALIDATION.md)
