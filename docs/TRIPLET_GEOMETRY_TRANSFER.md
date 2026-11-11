# ORCA Nova — Triplet Geometry Transfer + Spin (시험 기능)

## 목적 / 사용 대상

S-alkyl thiouracil 유도체의 **Reference T₁ 고리 구조를 Target에 공통 시작 구조로 이식**하고, 초기 구조가 다른 UKS Triplet 최적화의 Spin Population에 미치는 영향을 검사합니다. 이는 다른 치환기의 값이 같아지도록 조정하는 도구가 아닙니다. **완료된 UKS 계산은 최저 T₁ 또는 동일 전자배치의 증명이 아닙니다.**

기존 `W2`, `W5`, 일반 INP Editor, 다중 XYZ Batch 및 업데이트 로직은 그대로 유지합니다. 새 프리셋은 기존 W2를 변경하지 않는 **별도 Triplet 전용 패널**을 엽니다.

## 사용 순서

1. 새 프리셋 **Triplet Geometry Transfer + Spin**을 선택합니다.
2. 이미 최적화된 Triplet Reference XYZ 하나와 Target XYZ 1~24개를 불러옵니다. 3D 미리보기에서 각 원자 0-based index를 확인할 수 있습니다.
3. **① Mapping·Geometry 미리보기**를 누르면 고리 `C2–N3–C4–C5–C6–N1` 및 `C2–S2`, `C4–O4` 결합 그래프를 기준으로 원자 역할을 탐지합니다.
4. Reference/Target의 원자 인덱스가 맞는지 확인합니다. 자동 탐지가 모호하거나 실패하면 **역할별 인덱스 직접 입력** 후 **② 수동 매핑 재검증**을 실행합니다.
5. 기본 고정 원자는 고리 heavy atoms 6개입니다. 필요시 `S2`, `O4`, `H1/H3/H5/H6`도 고정할 수 있습니다. 해당 H가 두 XYZ 모두에 존재하고 그 부모 고리에 결합되어야 합니다.
6. Reference/Target Charge(+1 N1-protonated 예시, 0 중성), Frequency, Method, Basis, Solvent, 각 Job CPU/MaxCore 및 전체 Batch CPU/RAM/동시실행 수를 확인합니다.
7. **③ 확인 후 Batch 등록**을 누르면 독립적인 작업 디렉터리에서 실행됩니다. 기존 Batch Scheduler의 Preflight와 예약·취소·재시도 로직을 사용합니다.
8. **Queue 결과 새로고침 → Spin/Bond 결과 분석**에서 C2/C5/C6, 결합 길이·총 에너지·⟨S²⟩·SCF/Opt/Frequency·CUBE·GBW를 확인할 수 있습니다.
9. 동일한 Target에서 얻은 Original W2의 OUT/INP/최종 XYZ를 넣어 **Original / Transfer 비교**를 수행합니다. 구조·화학식·연결성·Method/Charge/Spin·수렴을 검사한 후에만 동일 Target의 `ΔE`를 `kcal/mol`로 표시합니다. 원본 비교 CSV/JSON도 저장할 수 있습니다. S₀ 에너지는 **S₀ OUT/INP/최종 XYZ가 모두 있고 검증될 때만** 추가 출력합니다.

## 작업별 입력/산출물

| 단계 | 입력 | 결과 |
|---|---|---|
| STEP 0: Atom Mapping &amp; XYZ Transfer | Ref + Target XYZ | `transferred_initial.xyz` (ORCA 실행 아님) |
| STEP 1: Constrained Opt | `step1_constrained.inp` + transferred XYZ | `step1_constrained.out`, `step1_constrained.xyz` |
| STEP 2: Unconstrained Opt | `step2_triplet.inp` + STEP 1 XYZ | `step2_triplet.out`, `step2_triplet.xyz`, `step2_triplet.gbw` |
| STEP 3: Spin Density SP | `step3_spin.inp` + STEP 2 XYZ | `step3_spin.out`, `step3_spin.cube`, Atomic Spin Populations |
| Report | Stage 2/3 files, confirmed role mapping | `triplet_spin_report.json`, `triplet_spin_report.csv` (값이 생성된 경우에만) |

**ORCA 입력 예시 — STEP 1**

```text
! UKS B3LYP D4 def2-TZVP TightSCF Opt SMD(acetonitrile)
%pal
 nprocs 8
end
%maxcore 2500
%geom
 Constraints
  { C 0 C }
  { C 4 C }
  ...
 end
end
* xyzfile 1 3 transferred_initial.xyz
```

각 제약 index는 Target의 **실제 chemical role mapping에 따라 자동 생성**됩니다. STEP 2에는 `%geom Constraints`가 존재하지 않습니다. STEP 3에는 STEP 2에서 생성한 `step2_triplet.gbw`를 `! MOREAD`와 `%moinp "step2_triplet.gbw"`로 불러와 **이전 UKS orbitals를 SCF 초기값으로 사용**합니다. 같은 Job 폴더의 승인된 STEP 2 GBW만 허용하며 일반 Batch의 외부 `%moinp`/MOREAD 차단은 유지합니다. 또한 ORCA 6용 `%output Print[P_Mulliken/P_Loewdin/P_Hirshfeld]`와 `%plots Format Gaussian_Cube, SpinDens("step3_spin.cube")`가 들어갑니다. **MOREAD는 전자상태가 유지되었다는 증명이 아니며**, STEP 2·3의 Occupation/⟨S²⟩/Spin 분포를 비교해야 합니다.

[ORCA 6.1 MORead/GBW 공식 매뉴얼](https://www.faccts.de/docs/orca/6.1/manual/contents/essentialelements/initialguess.html) · [ORCA 6.1 Geometry Constraints 공식 매뉴얼](https://www.faccts.de/docs/orca/6.1/manual/contents/structurereactivity/optimizations.html) · [ORCA Population Analysis](https://www.faccts.de/docs/orca/6.1/manual/contents/spectroscopyproperties/population.html) · [ORCA Spin Density Cube](https://www.faccts.de/docs/orca/6.1/manual/contents/utilitiesvisualization/plots.html)

## 과학적 한계와 검증 경계

- **XYZ 파일에는 Bond Order/Charge/Multiplicity/Protonation 선언이 내재하지 않습니다.** 결합 연결성은 공유결합 반지름 기반 추정이며, C2–S, C4–O, N1/N3 및 고리 수소 개수와 선언된 charge를 검사합니다. Tautomer, 실제 양이온 성격, 특정 T₁ 상태의 wavefunction은 원본 OUT/구조로 검토해야 합니다.
- **구조 재사용은 초기 좌표를 통일하는 것일 뿐, 다른 Triplet 최소점이 존재하는지 증명하지 않습니다.** 필요시 서로 다른 초기 Reference, broken-symmetry/SCF guess, alternative orbital occupation, Frequency 등 별도 계산이 필요합니다.
- `Frequency OFF`면 **최소점 미검증**으로 표시합니다. 켜면 STEP 2의 허수 모드 수가 0인지 보고서에서 판단합니다. `⟨S²⟩`가 Triplet의 기대값 2.0과 크게 다르면 경고합니다.
- Triplet SOMO 후보는 alpha/beta orbital occupation을 기반으로 *후보*만 기록하며 실제 공간 궤도 성격·spin coupling을 자동 확증하지 않습니다. `UNO/UCO` 같은 별도 검토가 도움될 수 있습니다.
- **서로 다른 유도체 사이 절대 Electronic Energy는 안정성 비교에 사용하지 않습니다.** 에너지 차는 **동일 Target**의 Original vs Transfer에서 Method·전자상태·수렴·연결성이 검증된 경우에만 수치화합니다.
- Trial workflow는 **정식 Stable 배포되지 않았고, 실제 연구실 Windows ORCA 6.x/MPI로 재계산된 정답 검증이 필요합니다.** GitHub Windows 빌드·Mock 계산 통과는 **real ORCA scientific run**과 다릅니다.
- 매핑은 현재 **C2–S / C4–O가 있는 thiouracil pyrimidine 고리**를 대상으로 합니다. 축합고리·교차 결합·잘못 표기된 수소/원소·서로 다른 protonation 위치는 거절하거나 수동 검증이 필요합니다. 일반적인 임의 분자 scaffold matching 도구가 아닙니다.

## 테스트 및 릴리스 제한

`triplet_transfer_test.go`: 0-based mapping/permutation, protonation and charge guards, 고리 이식 및 Constraints.

`triplet_queue_test.go`: 다중 Target Mock 병렬 Batch, 단계 좌표 전달, 실패 시 원인 분석, CUBE/GBW, 역할별 Spin, 다른 화학식의 ΔE 거부.

`tests/triplet-transfer-ui.cjs`: 실제 Chromium GUI에서 파일 입력·Atom Mapping 수동 확인·사용자 승인·Batch 예산·Spin 비교.

연구실 실제 ORCA 실행으로 STEP 1/2/3 수렴, `step3_spin.cube` 생성, 실제 ⟨S²⟩·C2/C5/C6·Hirshfeld/Löwdin 값·SOMO 후보 및 Frequency를 확인해야 **정식판 안정성 검증**이 됩니다. 데이터는 로컬에 남고 GitHub로 자동 업로드되지 않습니다.
