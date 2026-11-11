# Windows CI self-test. Does not execute proprietary ORCA or modify personal data.
# Checks that -Execute refuses to overwrite existing research outputs.
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$root=Join-Path ([IO.Path]::GetTempPath()) ("orca-triplet-audit-safety-"+[guid]::NewGuid().ToString("N"))
$work=Join-Path $root 'job';$exe=Join-Path $root 'orca.exe'
try{
 New-Item -ItemType Directory -Force -Path $work|Out-Null
 Copy-Item -LiteralPath (Get-Command 'where.exe' -CommandType Application).Source -Destination $exe
 $inp=@'
! UKS B3LYP D4 def2-TZVP TightSCF Opt SMD(acetonitrile)
%geom
 Constraints
  { C 0 C }
 end
end
* xyzfile 1 3 transferred_initial.xyz
'@
 $xyz=@'
3
dummy only
C 0.000 0.000 0.000
N 1.250 0.000 0.000
S -1.700 0.000 0.000
'@
 Set-Content -LiteralPath (Join-Path $work 'step1_constrained.inp') -Value $inp -Encoding ascii
 Set-Content -LiteralPath (Join-Path $work 'transferred_initial.xyz') -Value $xyz -Encoding ascii
 $prior=Join-Path $work 'step1_constrained.out'
 $original="PRIOR ORIGINAL RESEARCH RESULTS - NEVER OVERWRITE"
 Set-Content -LiteralPath $prior -Value $original -Encoding utf8
 $hashBefore=(Get-FileHash -LiteralPath $prior -Algorithm SHA256).Hash
 & pwsh -NoProfile -NonInteractive -File (Join-Path $PSScriptRoot 'Windows-Triplet-RealORCA-Acceptance.ps1') -OrcaExe $exe -JobFolders $work -Execute
 if($LASTEXITCODE -ne 1){throw "Expected a fail-closed exit code 1 when OUT already exists"}
 $hashAfter=(Get-FileHash -LiteralPath $prior -Algorithm SHA256).Hash
 if($hashBefore -ne $hashAfter){throw "Existing research output was overwritten"}
 $reportFile=Join-Path $work 'Triplet_RealORCA_Acceptance.json'
 if(-not(Test-Path -LiteralPath $reportFile)){throw "Missing local acceptance JSON"}
 $report=Get-Content -LiteralPath $reportFile -Raw | ConvertFrom-Json
 if($report.Passed -ne $false){throw "Fail-closed test was improperly marked as passed"}
 if(-not (@($report.Issues)|Where-Object {$_ -match 'overwritten'})){throw "Overwrite refusal not identified in scientific audit"}
 if(-not(Test-Path -LiteralPath (Join-Path $work 'BUILD_FAILURE_ANALYSIS.md'))){throw "Missing failure root cause report"}
 Write-Host 'PASS: existing OUT/XYZ are preserved, execution refused, error report local-only'
}finally{
 if(Test-Path -LiteralPath $root){Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue}
}
