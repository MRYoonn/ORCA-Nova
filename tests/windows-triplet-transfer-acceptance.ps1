<#
Local-only acceptance for an actual Windows ORCA6 Triplet Geometry Transfer job.
No data leaves the user's computer. Does not by itself prove the T1 identity.
#>
[CmdletBinding()]
param([Parameter(Mandatory=$true)][string]$Folder,[switch]$RequireFrequency)
$ErrorActionPreference='Stop'
$Root=(Resolve-Path -LiteralPath $Folder).Path
$checks=[ordered]@{}
$errors=[System.Collections.Generic.List[string]]::new()
function Check([string]$label,[bool]$ok,[string]$reason){
 $checks[$label]=$ok
 if(-not $ok){$errors.Add($label+': '+$reason)}
}
function ReadLocal([string]$name){
 $path=Join-Path $Root $name
 if(-not (Test-Path -LiteralPath $path -PathType Leaf)){return ''}
 return [System.IO.File]::ReadAllText($path)
}
function HasLocal([string]$name){
 $p=Join-Path $Root $name
 return ((Test-Path -LiteralPath $p -PathType Leaf) -and (Get-Item -LiteralPath $p).Length -gt 0)
}
foreach($base in @('step1_constrained','step2_triplet','step3_spin')){
 Check ($base+'.inp') (HasLocal ($base+'.inp')) 'INP missing'
 Check ($base+'.out') (HasLocal ($base+'.out')) 'OUT missing'
 $out=ReadLocal ($base+'.out')
 Check ($base+' normal') ($out -match 'ORCA TERMINATED NORMALLY' -and $out -notmatch 'ERROR TERMINATION') 'normal ORCA ending missing'
 Check ($base+' SCF') ($out -match 'SCF CONVERGED' -and $out -notmatch 'SCF NOT CONVERGED|SCF DID NOT CONVERGE') 'SCF not converged'
 Check ($base+' energy') ($out -match 'FINAL SINGLE POINT ENERGY\s+[-+]?\d+\.\d+') 'no final energy'
 if($base -ne 'step3_spin'){
  Check ($base+' Opt') ($out -match 'THE OPTIMIZATION HAS CONVERGED') 'Opt did not converge'
  Check ($base+'.xyz') (HasLocal ($base+'.xyz')) 'optimized XYZ missing'
 }
 Check ($base+' UKS') ((ReadLocal ($base+'.inp')) -match '(?im)^\s*!\s*UKS\b') 'UKS missing'
 Check ($base+' multiplicity') ((ReadLocal ($base+'.inp')) -match '(?im)\*\s*xyzfile\s+[-+]?\d+\s+3\s+') 'triplet mult=3 missing'
}
$fixed=ReadLocal 'step1_constrained.inp'
$free=ReadLocal 'step2_triplet.inp'
$spin=ReadLocal 'step3_spin.inp'
Check 'STEP 0 transferred' (HasLocal 'transferred_initial.xyz') 'transferred_initial.xyz missing'
Check 'STEP 1 constraints' ($fixed -match '(?s)%geom\s+Constraints.*?\{\s*C\s+\d+\s+C\s*\}') 'Cartesian constraints not found'
Check 'STEP 1 geometry source' ($fixed -match '(?i)\*\s*xyzfile\s+[-+]?\d+\s+3\s+transferred_initial\.xyz') 'wrong input'
Check 'STEP 2 no constraints' ($free -notmatch '(?i)%geom\s+Constraints') 'free Opt contains constraints'
Check 'STEP 2 source' ($free -match '(?i)\*\s*xyzfile\s+[-+]?\d+\s+3\s+step1_constrained\.xyz') 'not chained'
Check 'STEP 3 source' ($spin -match '(?i)\*\s*xyzfile\s+[-+]?\d+\s+3\s+step2_triplet\.xyz') 'not chained'
Check 'Triplet GBW' (HasLocal 'step2_triplet.gbw') 'GBW not generated'
Check 'Spin Density Cube' (HasLocal 'step3_spin.cube') 'CUBE not generated'
$spinout=ReadLocal 'step3_spin.out'
Check 'Mulliken Spin' ($spinout -match '(?i)MULLIKEN ATOMIC CHARGES') 'Mulliken missing'
Check 'Loewdin Spin' ($spinout -match '(?i)LOEWDIN ATOMIC CHARGES|LÖWDIN ATOMIC CHARGES') 'Loewdin missing'
Check 'Hirshfeld Spin' ($spinout -match '(?i)HIRSHFELD ANALYSIS|HIRSHFELD ATOMIC CHARGES') 'Hirshfeld missing'
Check 'S2 value' ($spinout -match '(?i)<S(?:\*\*|\^)?2>\s*(?:\:|=)\s*[-+]?\d+\.?\d*') 'S2 missing'
$hasFreq=($free -match '(?i)\bFreq\b')
if($RequireFrequency){Check 'Frequency requested' $hasFreq 'Frequency missing from Opt'}
if($hasFreq){Check 'Frequency output' ((ReadLocal 'step2_triplet.out') -match 'VIBRATIONAL FREQUENCIES') 'frequency not found'}
if(HasLocal 'step3_spin.cube'){
 $header=@(Get-Content -LiteralPath (Join-Path $Root 'step3_spin.cube') -TotalCount 6)
 Check 'Gaussian Cube header' ($header.Count -ge 6 -and $header[2] -match '^\s*[-+]?\d+\s+[-+]?\d') 'Cube grid header is not valid'
}
if(HasLocal 'triplet_spin_report.json'){
 try{
  $data=(ReadLocal 'triplet_spin_report.json')|ConvertFrom-Json -ErrorAction Stop
  $roles=@($data.spin|ForEach-Object{$_.role})
  Check 'C2/C5/C6 results' (('C2' -in $roles) -and ('C5' -in $roles) -and ('C6' -in $roles)) 'C2/C5/C6 spin missing'
 }catch{$errors.Add('Triplet analysis JSON parse failed: '+$_.Exception.Message)}
}else{$errors.Add('Triplet spin report missing')}
$result=[ordered]@{
 Application='ORCA Nova Triplet Geometry Transfer'
 Created=(Get-Date).ToString('o')
 LocalFolder=$Root
 AllChecksPassed=($errors.Count -eq 0)
 FrequencyRequested=$hasFreq
 LowestTripletT1Identity='Unverified even if all artifact checks pass'
 Checks=$checks
 Problems=@($errors)
}
$outFile=Join-Path $Root 'TRIPLET_ORCA6_ACCEPTANCE.json'
$result|ConvertTo-Json -Depth 8|Set-Content -LiteralPath $outFile -Encoding UTF8
if($errors.Count){
 $report=([string]::Join([Environment]::NewLine,@('# Triplet ORCA 6 actual run: FAIL','',($errors|ForEach-Object{'- '+$_}),'','Keep all original OUT/INP/XYZ/GBW/CUBE files.')))
 Set-Content -LiteralPath (Join-Path $Root 'TRIPLET_FAILURE_ANALYSIS.md') -Value $report -Encoding UTF8
 Write-Host ('FAIL: '+$errors.Count+' issues. Read '+$outFile)
 exit 1
}
Write-Host ('PASS: stage/artifact acceptance. T1 identity still unverified. '+$outFile)
exit 0
