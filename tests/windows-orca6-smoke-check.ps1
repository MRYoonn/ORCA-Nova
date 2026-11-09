# Local ORCA 6.x / ORCA Nova research-PC acceptance audit.
# Run after executing the W2 (S0 Opt+Freq -> T1 Opt+Spin) workflow on Windows.
# This script never uploads a molecule, OUT, GBW, CUBE, or report anywhere.
param(
 [Parameter(Mandatory=$true)][string]$OrcaExe,
 [Parameter(Mandatory=$true)][string]$JobFolder,
 [switch]$RequireBothStates,
 [switch]$RequireCube,
 [switch]$RequireFrequency,
 [switch]$RequireGBW
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$issues=[System.Collections.Generic.List[string]]::new()
$stages=[System.Collections.Generic.List[object]]::new()
if (-not (Test-Path -LiteralPath $OrcaExe -PathType Leaf)) {
 Write-Error "ORCA executable not found: $OrcaExe";exit 2
}
if ([IO.Path]::GetFileName($OrcaExe) -ine 'orca.exe') {
 Write-Error 'The executable must be named orca.exe';exit 2
}
if (-not (Test-Path -LiteralPath $JobFolder -PathType Container)) {
 Write-Error "Calculation folder not found: $JobFolder";exit 2
}
$inputs=@(Get-ChildItem -LiteralPath $JobFolder -File -Filter '*.inp')
if ($inputs.Count -eq 0) {$issues.Add('No stage INP files in the calculation folder.')}
$seenSinglet=$false
$seenTriplet=$false
foreach ($inpFile in $inputs) {
 $inp=Get-Content -LiteralPath $inpFile.FullName -Raw -ErrorAction Stop
 $output=Join-Path $JobFolder ($inpFile.BaseName+'.out')
 $isOpt=[regex]::IsMatch($inp,'(?im)^\s*!\s*.*\b(?:Opt|OptTS)\b')
 $isFreq=[regex]::IsMatch($inp,'(?im)^\s*!\s*.*\b(?:Freq|NumFreq|AnFreq)\b')
 $mult=[regex]::Match($inp,'(?im)^\s*\*\s*xyzfile\s+-?\d+\s+(\d+)')
 if ($mult.Success) {
  if ($mult.Groups[1].Value -eq '1') {$seenSinglet=$true}
  if ($mult.Groups[1].Value -eq '3') {$seenTriplet=$true}
 }
 $state=[ordered]@{Stage=$inpFile.Name;OUT=([IO.Path]::GetFileName($output));Normal=$false;SCF=$false;Optimization=$null;Frequency=$null;GBW=$null}
 if (-not (Test-Path -LiteralPath $output -PathType Leaf)) {
  $issues.Add("Missing OUT for stage $($inpFile.Name)")
  $stages.Add($state)
  continue
 }
 $outText=Get-Content -LiteralPath $output -Raw
 $state.Normal=$outText.Contains('ORCA TERMINATED NORMALLY')
 $state.SCF=$outText.Contains('SCF CONVERGED')
 if ($isOpt) {$state.Optimization=$outText.Contains('THE OPTIMIZATION HAS CONVERGED')}
 if ($isFreq) {$state.Frequency=$outText.Contains('VIBRATIONAL FREQUENCIES')}
 if ($RequireGBW) {$state.GBW=Test-Path -LiteralPath (Join-Path $JobFolder ($inpFile.BaseName+'.gbw')) -PathType Leaf}
 if (-not $state.Normal) {$issues.Add("ORCA normal termination missing: $($inpFile.Name)")}
 if (-not $state.SCF) {$issues.Add("SCF convergence missing: $($inpFile.Name)")}
 if ($isOpt -and -not $state.Optimization) {$issues.Add("Opt convergence missing: $($inpFile.Name)")}
 if ($isFreq -and -not $state.Frequency) {$issues.Add("Frequency output missing: $($inpFile.Name)")}
 if ($RequireGBW -and -not $state.GBW) {$issues.Add("GBW missing: $($inpFile.Name)")}
 if ($isOpt -and -not (Test-Path -LiteralPath (Join-Path $JobFolder ($inpFile.BaseName+'.xyz')) -PathType Leaf)) {
  $issues.Add("Optimized XYZ missing: $($inpFile.Name)")
 }
 $stages.Add($state)
}
if ($RequireBothStates -and -not ($seenSinglet -and $seenTriplet)) {
 $issues.Add('The supplied INP files do not include both Singlet and Triplet multiplicities.')
}
if ($RequireFrequency -and -not (@($stages | Where-Object {$_.Frequency -eq $true}).Count -gt 0)) {
 $issues.Add('No Frequency result was confirmed.')
}
$cubeFiles=@(Get-ChildItem -LiteralPath $JobFolder -File | Where-Object {$_.Extension -in @('.cube','.cub')})
if ($RequireCube -and $cubeFiles.Count -eq 0) {
 $issues.Add('Requested Spin Density cube is missing (.cube/.cub).')
}
$report=[ordered]@{
 Audit='ORCA Nova local Windows ORCA 6.x output acceptance'
 AuditTime=(Get-Date).ToString('o')
 ExecutablePathExists=[bool](Test-Path -LiteralPath $OrcaExe -PathType Leaf)
 Check='Evidence inspection only; this audit does not run ORCA or validate chemical interpretation.'
 JobFolder=$JobFolder
 Stages=$stages.ToArray()
 CubeFiles=@($cubeFiles | ForEach-Object {$_.Name})
 Passed=($issues.Count -eq 0)
 Issues=$issues.ToArray()
 HumanVisualChecks=@('ORCA Nova native app opens without errors',
 'Orca icon visible in taskbar/window titlebar',
 'Batch separate jobs use distinct folders',
 'Cancelling one ORCA/MPI job leaves the others running',
 'No surviving child orca.exe/mpiexec.exe after cancellation',
 'Updating while a job runs is blocked; approved stable update restarts and preserves settings')
}
$destination=Join-Path $JobFolder 'Windows_ORCA6_Smoke_Audit.json'
$report | ConvertTo-Json -Depth 7 | Set-Content -LiteralPath $destination -Encoding UTF8
Write-Host ("LOCAL_QA_PASS={0}" -f $report.Passed)
Write-Host ("STAGES={0}, CUBE_FILES={1}, ISSUES={2}" -f $stages.Count,$cubeFiles.Count,$issues.Count)
Write-Host ("LOCAL_ONLY_REPORT={0}" -f $destination)
foreach ($issue in $issues) {Write-Host ("FAIL: {0}" -f $issue)}
if ($issues.Count -gt 0) {exit 1}
exit 0
