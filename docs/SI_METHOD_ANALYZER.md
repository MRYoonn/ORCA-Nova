# ORCA Nova · SI 기반 계산 Method 분석 (로컬, 실험적)

## 기능과 검증 경계

ORCA Nova는 이제 **텍스트 복사가 가능한 SI PDF**를 로컬 API로 불러오고 계산 조건 후보의 **PDF 페이지 번호·근거 텍스트·DOI(사용자 입력)**를 제시합니다. 현재 INP와 대조한 뒤 **사용자가 체크한 지원 옵션만 확인창을 거쳐 적용**합니다. Gaussian 전용/지원되지 않는 Basis, 여러 Method가 혼합된 SI, 정확히 어느 계산 단계의 조건인지 미확정인 값은 자동 적용하지 않습니다.

**한계:** PDF 텍스트 추출은 오래된 경량 Go `rsc.io/pdf` 기반의 best-effort입니다. 최신 xref/폰트/CMap, 이미지 기반 SI 및 복잡한 배치에서 텍스트가 깨지거나 추출이 실패할 수 있습니다. OCR, DOI 자동 다운로드·진위 검증, 모든 SI Method 자동 파악/재현, 사용자 실제 ORCA/MPI 재계산은 제공하지 않습니다. 이때는 Computational Methods 텍스트를 직접 입력하는 폴백을 사용합니다.

파일은 **127.0.0.1 로컬 세션 인증 API**에서만 처리하며 DOI가 외부 서버에 전송되거나 PDF가 자동 업로드/보관되지 않습니다. PDF 입력 크기는 12MB, 텍스트는 2MB 이하입니다. 개인정보/미발표 데이터가 포함된 원본을 공개 GitHub에 넣지 마세요.

## 사용 방법

1. `v1.6.0-rc.7` 이후 최신 시험판에서 **Supporting Information · PDF Method 분석 / INP 대조**를 펼칩니다.
2. SI PDF를 불러오거나 SI의 **Computational/DFT Method** 문단을 직접 붙여넣고 **DOI**를 기록합니다.
3. **PDF / SI 분석 및 현재 INP 비교** 버튼을 눌러 Functional/Basis/Dispersion/Solvent/Software/Task 후보와 페이지·발췌문을 확인합니다.
4. **Ambiguous, Unsupported, Gaussian** 후보는 수동 검토합니다. **지원되는 고유 후보**만 체크하고 확인 창에서 적용합니다. **Opt/Freq와 SP가 서로 다른 수준**이면 각각 독립된 INP/Workflow로 설정합니다.
5. 결과에 오류가 있으면 **근거 TXT**를 저장해 검토합니다. 이 파일은 문헌 조건을 검증했다는 증명이 아닙니다.

## Opt + Freq / SP Composite Gibbs

6개 필수 파일:
- 저수준 `Opt + Freq` INP/OUT, 해당 Opt 결과의 최종 XYZ
- 고수준 `SP` INP/OUT, SP에 실제 입력한 XYZ

수렴된 파일을 확인한 뒤 다음 식을 사용합니다.

```text
G_composite (Eh) = E_high,SP (Eh) + [G_low,Opt+Freq (Eh) - E_low,Opt+Freq (Eh)]
```

동일 화학종·원자 순서/내부 거리(허용차 0.002 Å)·Charge·Multiplicity·Solvation Model·Solvent가 일치해야 하며, `Minimum`은 imaginary=0, `TS`는 imaginary=1을 요구합니다. SP는 Geometry를 최적화하지 않아야 합니다. 충족하지 않으면 계산을 중단합니다.

결과는 **한 화학종의 혼합수준 절대 Gibbs Energy (Eh)**입니다. 독립 화학종 사이 반응/활성화 자유에너지 `ΔG` 또는 `ΔG‡` (kcal/mol)를 자동 산출하지 않습니다. 화학양론, Electronic State, 표준상태(1 atm/1 M), 실험 용매, quasi-harmonic/저주파 보정, 상대적인 Reference 상태를 별도로 검증해야 합니다. 구조가 다르면 기하학적으로 정렬해도 비교를 거절합니다.

## 독립된 공개 SI 3건의 손대조 자료

다음 자료는 PDF/원문 계산 Method에 표시된 **대표 키워드**를 수동으로 정리하여 `si_analysis_test.go`에서 텍스트 기반 회귀 사례로 재검증합니다. 전체 논문 데이터를 재현했다는 뜻은 아니며 PDF 원본 3종을 완전 자동 분석해 검증했다는 뜻도 아닙니다.

| SI | 페이지 | Method 핵심 | 주의 |
|---|---|---|---|
| Jiang et al., [phosphazenium salt SI](https://files.sciengine.com/gridfs/1828685594578911232/edit_SI.pdf) | S25 | Gaussian16, M06-2X, Opt: 6-31+G(d,p); SP: 6-311+G(d,p); SMD, Freq, Shermo | 2개 Basis를 하나로 합치지 않음; Gaussian Basis 직접 ORCA 입력 불가 |
| Hollister et al., [Chemical Science SI](https://www.rsc.org/suppdata/d4/sc/d4sc03835e/d4sc03835e1.pdf) | p70 | ORCA5.0.4, B3LYP-D3(BJ)/def2-TZVP, CPCM(Et2O/o-DFB), Frequency; 별도 CAM-B3LYP TD-DFT | Opt와 Vertical 기능/Method 혼동 금지 |
| Cui et al., [Angew SI](https://pure.mpg.de/pubman/item/item_3679013_2/component/file_3685440/467_anie70244-sup-0001-suppmat.pdf) | S31 | ORCA6, B3LYP D4 def2-TZVP CPCM(toluene) TightOpt TightSCF DefGrid3 Freq | 실제 원소·Charge·Spin과 TS 등 별도 확인 |

Glorius–Hartmann 2026 (DOI 10.1002/anie.9565676)의 기존 두 프리셋은 **공개 논문 본문 Figure 5의 Method 기록**에 기반합니다. **전체 SI 원문과 ioChem-BD 계산 파일 교차검증은 미완료**이므로 현재 코드에서도 문헌 전체를 재현했다고 표시하지 않습니다.

## 자동 QA

- `TestThreePublicSIMethodFixturesRequireHumanReview`: 문헌 3건에서 추출한 짧은 키워드 사례가 중복 Method를 모호함으로 표시하고 잘못 자동 적용하지 않는지 검증
- `TestDigitallyBornPDFFixturePerPageExtraction`: 독자 생성 디지털 PDF에서 페이지별 텍스트 추출
- `TestSIAPISessionProtectionAndBadPDF`: 로컬 API 세션 보호/손상 PDF 거절
- `TestCompositeGibbsValidations`: Opt/SP 계산 상태, 기하 구조·Charge/Multiplicity·Solvent·Imaginary Mode 검증
- `tests/si-ui.cjs`: 브라우저 내 SI 파일·텍스트 입력, 근거/비지원 항목 차단, 사용자 확인창, Composite Gibbs 입력과 결과

**실제 연구 논문의 결론/메커니즘을 재현하려면 별도의 독립 계산과 과학적 검증이 반드시 필요합니다.**
