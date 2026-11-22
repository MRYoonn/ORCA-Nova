# ORCA Nova v1.8.0-rc.2

Windows용 Beta 후보입니다. 앱의 Settings에서 Beta 채널을 선택하면 공개된 시험판을 검색할 수 있습니다.

- Glorius 2026 SI 기반 프리셋 13개를 제공합니다. GOAT 단계는 `.globalminimum.xyz`를 전달하고, Minimum과 TS는 각각 0개와 1개의 imaginary mode를 확인합니다.
- Composite Gibbs는 Opt/Freq와 High-level SP가 실행 검사를 통과하고, SP 입력이 해당 Optimization의 XYZ를 참조할 때 `analysis.txt`와 `results.json`에 기록됩니다.
- Calculation의 Run 옆과 Batch 및 Results에서 결과 폴더를 열 수 있습니다. Batch와 Results에서는 경로를 클립보드에 복사할 수 있습니다.
- Calculation에서 현재 단계와 전체 Workflow ETA를 표시합니다. Batch ETA 범위는 지난 실행시간과 CPU, RAM 및 동시 실행 한도를 반영합니다. 유사 기록이 부족하거나 실행시간이 예상 범위를 넘으면 불확실 상태를 표시합니다.

Glorius 계산 조건은 SI에서 재구성했습니다. 원본 INP 대조와 실제 ORCA 6.x/xTB Workflow 재현 검증은 남아 있습니다. TS의 imaginary mode와 IRC 종점, T1의 전자 상태 및 spin contamination은 연구자가 확인해야 합니다.

[Glorius Workflow와 검증 범위](GLORIUS_2026_SI_WORKFLOW.md)
