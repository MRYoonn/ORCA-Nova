# kcal/mol 중심 에너지 표시 (2026-11-10)

- 사용자가 익숙한 **kcal/mol**을 **Relative Energy** 그래프의 기본값으로 지정합니다. `viz-unit` 선택창에서 Eh / kJ/mol / eV로 다시 변경할 수 있습니다.
- 검증된 `S₀/T₁ Adiabatic` 및 `Reaction R–TS–P` 프로파일의 **ΔE, ΔG, activation barrier**를 **kcal/mol 먼저**, kJ/mol을 괄호에 함께 표시합니다. 프로파일 그래프 Y축도 kcal/mol로 표시합니다.
- **백엔드의 독립 검증된 원본 kJ/mol API는 바꾸지 않습니다.** 프런트엔드에서만 1 kcal = 4.184 kJ로 변환합니다. 원본 점의 `rawEnergyKJ`, `rawEnergyEh` 출처 정보는 유지됩니다.
- **TD-DFT Vertical Excitation**의 표시 테이블과 Batch 분자별 비교는 kcal/mol 먼저, eV / nm / oscillator strength를 병기합니다. Root 그래프는 기본 kcal/mol이며 사용자 선택에 따라 eV, kJ/mol, Eh로 바꿀 수 있습니다. 1 eV = 23.060548867 kcal/mol; UV-Vis 분석용 eV를 지우지 않습니다.
- **절대 Electronic/Gibbs/Enthalpy와 ZPE 보정항**은 ORCA의 Eh 그대로 유지합니다. 동일하거나 비교가능한 계산 조건, 상태, 구조 및 화학양론이 확인되기 전에는 분자 간 총 에너지 차이를 자동으로 의미있는 ΔG, ΔE로 해석하지 않습니다.
- 결과 요약의 고급 JSON과 원본 OUT는 수정 없이 조회할 수 있습니다. 값이 없는 경우 Not calculated를 유지합니다.
- `tests/batch-presets-summary-ui.cjs`에서 기본 단위, 각 변환 값, 원본 보존과 이중 단위 표시를 검증합니다.

예시 (가상값): `E(S0) = -482.123457 Eh`, `E(T1) = -482.041123 Eh`일 때 `ΔE = +51.67 kcal/mol` (약 `+2.24 eV`). 전자 에너지 차이와 Gibbs 자유에너지 차이를 혼용하지 않습니다.
