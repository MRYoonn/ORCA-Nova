# Research-PC-only acceptance test for Triplet Geometry Transfer + Spin.
# No study XYZ, wavefunctions, INP/OUT, local reports, or molecular identifiers are uploaded.
# Without -Execute this audits only already existing files. -Execute actually runs licensed ORCA.
[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$OrcaExe,
 [Parameter(Mandatory=$true)][string[]]$JobFolders,
 [switch]$Execute,
 [int]$TimeoutSecondsPerStage=14400
)
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'
$allPassed=$true
if($JobFolders.Count -lt 1){throw "Specify one or more private Triplet target JobFolders"}
if(-not(Test-Path -LiteralPath $OrcaExe -PathType Leaf) -or [IO.Path]::GetFileName($OrcaExe) -ine 'orca.exe'){
 throw "A locally installed licensed orca.exe is required: $OrcaExe"
}
if($TimeoutSecondsPerStage -lt 60 -or $TimeoutSecondsPerStage -gt 604800){throw "TimeoutSecondsPerStage must be between 60 and 604800"}
$stages=@(
 @{Stem='step1_constrained';Input='step1_constrained.inp';Source='transferred_initial.xyz';Opt=$true},
 @{Stem='step2_triplet';Input='step2_triplet.inp';Source='step1_constrained.xyz';Opt=$true},
 @{Stem='step3_spin';Input='step3_spin.inp';Source='step2_triplet.xyz';Opt=$false}
)
foreach($job in $JobFolders){
 $issues=[System.Collections.Generic.List[string]]::new()
 $warnings=[System.Collections.Generic.List[string]]::new()
 $stageReport=[System.Collections.Generic.List[object]]::new()
 $folder=[IO.Path]::GetFullPath($job)
 if(-not(Test-Path -LiteralPath $folder -PathType Container)){
  Write-Warning "Calculation folder not found: $folder";$allPassed=$false;continue
 }
 foreach($stage in $stages){
  $stem=$stage.Stem;$inpPath=Join-Path $folder $stage.Input;$sourcePath=Join-Path $folder $stage.Source
  $outPath=Join-Path $folder ($stem+'.out')
  $xyzOut=Join-Path $folder ($stem+'.xyz')
  $state=[ordered]@{Stage=$stem;InputSource=$stage.Source;InputExists=$false;InputValid=$false;Run='NOT_REQUESTED';Normal=$false;SCF=$false;Opt=$null;OutputXYZ=$false;ElapsedSeconds=$null}
  if(-not(Test-Path -LiteralPath $inpPath -PathType Leaf)){ $issues.Add("${stem}: INP missing");$stageReport.Add($state);break }
  $state.InputExists=$true
  $inp=Get-Content -LiteralPath $inpPath -Raw
  $validMult=[regex]::IsMatch($inp,'(?im)^\s*\*\s*xyzfile\s+[+-]?\d+\s+3\s+')
  $validSource=[regex]::IsMatch($inp,'(?im)^\s*\*\s*xyzfile\s+[+-]?\d+\s+3\s+'+[regex]::Escape($stage.Source)+'\s*$')
  $validUKS=[regex]::IsMatch($inp,'(?i)^\s*!\s*.*\bUKS\b')
  $constraints=[regex]::IsMatch($inp,'(?is)%geom\s+Constraints\b')
  $mustConstrain=$stem -eq 'step1_constrained'
  $state.InputValid=$validMult -and $validSource -and $validUKS -and ($constraints -eq $mustConstrain)
  if(-not $state.InputValid){$issues.Add("${stem}: UKS Triplet / XYZ source / Cartesian constraints input does not match expected workflow")}
  if(-not(Test-Path -LiteralPath $sourcePath -PathType Leaf)){$issues.Add("${stem}: Required source XYZ missing: $($stage.Source)")}
  if($issues.Count -gt 0){$stageReport.Add($state);break}
  if($Execute){
   # Refuse to overwrite any old scientific results on disk.
   if(Test-Path -LiteralPath $outPath -PathType Leaf){
    $issues.Add("${stem}: existing OUT would be overwritten. Use a newly prepared, empty target JobFolder. No existing scientific data were changed.")
    $stageReport.Add($state);break
   }
   $stderrPath=Join-Path $folder ($stem+'.stderr.txt')
   $start=Get-Date
   $proc=$null
   try{
    $proc=Start-Process -FilePath $OrcaExe -ArgumentList @($stage.Input) -WorkingDirectory $folder -RedirectStandardOutput $outPath -RedirectStandardError $stderrPath -PassThru -ErrorAction Stop
    if(-not $proc.WaitForExit($TimeoutSecondsPerStage*1000)){
     & taskkill.exe /PID $proc.Id /T /F 2>$null|Out-Null
     $state.Run='TIMEOUT'
     $issues.Add("${stem}: timeout after $TimeoutSecondsPerStage seconds; ORCA/MPI tree terminated")
    }elseif($proc.ExitCode -ne 0){
     $state.Run="EXIT_$($proc.ExitCode)"
     $issues.Add("${stem}: ORCA exit code $($proc.ExitCode)")
    }else{$state.Run='EXIT_0'}
   }catch{
    $issues.Add("${stem}: ORCA process error: "+[string]$_.Exception.Message)
   }finally{
    $state.ElapsedSeconds=[math]::Round(((Get-Date)-$start).TotalSeconds,1)
    if($null -ne $proc){$proc.Dispose()}
   }
  }
  if(-not(Test-Path -LiteralPath $outPath -PathType Leaf)){
   $issues.Add("${stem}: OUT file missing")
   $stageReport.Add($state);break
  }
  $out=Get-Content -LiteralPath $outPath -Raw
  $state.Normal=$out.Contains('ORCA TERMINATED NORMALLY') -and -not $out.Contains('ERROR TERMINATION')
  $state.SCF=$out.Contains('SCF CONVERGED') -and -not([regex]::IsMatch($out,'(?i)SCF (?:NOT CONVERGED|DID NOT CONVERGE)|SCF CONVERGENCE FAILURE'))
  if($stage.Opt){
   $state.Opt=$out.Contains('THE OPTIMIZATION HAS CONVERGED')
   $state.OutputXYZ=Test-Path -LiteralPath $xyzOut -PathType Leaf
  }
  if(-not $state.Normal){$issues.Add("${stem}: ORCA normal termination missing")}
  if(-not $state.SCF){$issues.Add("${stem}: SCF convergence not verified")}
  if($stage.Opt -and -not $state.Opt){$issues.Add("${stem}: Geometry Optimization not converged")}
  if($stage.Opt -and -not $state.OutputXYZ){$issues.Add("${stem}: final optimized XYZ missing")}
  if($stem -eq 'step2_triplet'){
   $gbw=Join-Path $folder 'step2_triplet.gbw'
   if(-not(Test-Path -LiteralPath $gbw -PathType Leaf) -or (Test-Path -LiteralPath $gbw -PathType Leaf) -and (Get-Item -LiteralPath $gbw).Length -eq 0){
    $issues.Add("STEP 2: valid nonempty Triplet GBW missing")
   }
   if([regex]::IsMatch($inp,'(?i)\bFreq\b|\bNumFreq\b|\bAnFreq\b')){
    if(-not $out.Contains('VIBRATIONAL FREQUENCIES')){$issues.Add('STEP 2: Frequency requested but not printed')}
    $negative=@([regex]::Matches($out,'(?m)^\s*\d+\s*:\s*(-\d+(?:\.\d+)?)\s+cm\*\*-1'))
    if($negative.Count -gt 0){$warnings.Add("STEP 2: Negative frequencies observed; inspect translational/rotational modes and imaginary vibrational modes")}
   }else{
    $warnings.Add('STEP 2: Frequency not requested. Triplet minimum NOT verified.')
   }
  }
  if($stem -eq 'step3_spin'){
   $cube=Join-Path $folder 'step3_spin.cube'
   if(-not(Test-Path -LiteralPath $cube -PathType Leaf) -or (Test-Path -LiteralPath $cube -PathType Leaf) -and (Get-Item -LiteralPath $cube).Length -lt 200){
    $issues.Add('STEP 3: nonempty Gaussian_Cube spin density file missing or incomplete')
   }
   foreach($kind in @('MULLIKEN','LOEWDIN','HIRSHFELD')){
    if($out.ToUpperInvariant().IndexOf($kind) -lt 0 -and -not($kind -eq 'LOEWDIN' -and $out.Contains('LÖWDIN'))){
     $issues.Add("STEP 3: $kind atomic population output missing")
    }
   }
   $spin=[regex]::Match($out,'(?i)<S(?:\*\*|\^)?2>\s*(?:\:|=)\s*([-+]?\d+(?:\.\d+)?)')
   if(-not $spin.Success){$issues.Add('STEP 3: <S**2> not found')}
   else{
    $s2=[double]::Parse($spin.Groups[1].Value,[Globalization.CultureInfo]::InvariantCulture)
    if([math]::Abs($s2-2.0) -gt 0.10){$warnings.Add("STEP 3: Spin contamination possible, <S²>=$s2, expected 2.0")}
   }
   if(-not $out.Contains('FINAL SINGLE POINT ENERGY')){$issues.Add('STEP 3: Final Single Point Energy missing')}
  }
  $stageReport.Add($state)
  if($issues.Count -gt 0){break}
 }
 $reportFile=Join-Path $folder 'triplet_spin_report.json'
 if($issues.Count -eq 0 -and (Test-Path -LiteralPath $reportFile -PathType Leaf)){
  try{
   $result=Get-Content -LiteralPath $reportFile -Raw|ConvertFrom-Json
   $actual=@($result.spin|ForEach-Object {$_.role})
   foreach($label in @('C2','C5','C6')){if($label -notin $actual){$issues.Add("Triplet report missing chemically mapped $label")}}
   if(-not $result.converged){$issues.Add('Triplet report marks calculation as not converged')}
   if(-not $result.cubeAvailable -or -not $result.gbwAvailable){$issues.Add('Triplet report missing verified CUBE or GBW')}
  }catch{$issues.Add('Triplet JSON analysis failed: '+[string]$_.Exception.Message)}
 }else{
  $warnings.Add('Triplet software report missing or still incomplete; direct OUT checks do not prove mapped C2/C5/C6 spin')
 }
 $report=[ordered]@{
  Title='ORCA Nova Triplet Transfer real Windows ORCA6 local acceptance'
  Date=(Get-Date).ToUniversalTime().ToString('o')
  JobFolder=$folder
  ORCAExecutable=$OrcaExe
  Mode=$(if($Execute){'REAL_ORCA_EXECUTION'}else{'INSPECTION_OF_EXISTING_LOCAL_OUTPUTS'})
  Stages=$stageReport.ToArray()
  Passed=($issues.Count -eq 0)
  Issues=$issues.ToArray()
  Warnings=$warnings.ToArray()
  ChemicalStateProven=$false
  TripletMinimumProven=$false
  Notes='Termination and output-file checks do not prove that the converged UKS state is the lowest T1 state. Human validation and, when enabled, vibrational frequencies remain required.'
 }
 $destination=Join-Path $folder 'Triplet_RealORCA_Acceptance.json'
 $report|ConvertTo-Json -Depth 7|Set-Content -LiteralPath $destination -Encoding UTF8
 if($issues.Count -gt 0){
  $allPassed=$false
  $failure=Join-Path $folder 'BUILD_FAILURE_ANALYSIS.md'
  $lines=@('# Triplet Geometry Transfer — Real Windows ORCA6 validation failure', '',"Job: $folder",'', '## Detected issues')
  $lines+=@($issues|ForEach-Object {'- '+$_})
  $lines+=@('', '## Follow-up','- Check corresponding stage INP/OUT and ORCA6/MPI version.', '- Do not mark a Triplet state or minimum as validated from program completion alone.', '- Rerun in a NEW job folder to preserve previous output.')
  $lines|Set-Content -LiteralPath $failure -Encoding UTF8
 }
 Write-Host ("TRIPLET_LOCAL_ACCEPTANCE_PASS={0}  FOLDER={1}  ISSUES={2}" -f $report.Passed,$folder,$issues.Count)
 foreach($issue in $issues){Write-Host ("FAIL: "+$issue)}
 foreach($warning in $warnings){Write-Host ("WARN: "+$warning)}
}
if(-not $allPassed){exit 1}
exit 0
