# Isolated Windows Portable smoke test. Does not change existing settings.
# Optional -OrcaExe runs a real licensed ORCA 6.x water calculation via the app.
[CmdletBinding()]
param(
 [Parameter(Mandatory=$true)][string]$PortableZip,
 [Parameter(Mandatory=$true)][string]$ChecksumFile,
 [string]$ExpectedVersion="1.6.0-rc.2",
 [string]$OrcaExe="",
 [string]$ReportPath="",
 [int]$RealOrcaTimeoutSeconds=180,
 [switch]$KeepTemporaryFiles
)
Set-StrictMode -Version Latest
$ErrorActionPreference="Stop"
$report=[ordered]@{
 schema="orca-nova-windows-live-acceptance-v1"
 date=(Get-Date).ToUniversalTime().ToString("o")
 version_expected=$ExpectedVersion
 checksum_verified=$false
 packaged_files_verified=$false
 backend_started=$false
 session_restriction_verified=$false
 live_http_verified=$false
 preflight_verified=$false
 real_orca_execution="NOT_REQUESTED"
 passed=$false
 errors=@()
}
if(-not $ReportPath){$ReportPath=Join-Path (Get-Location).Path "ORCA_NOVA_WINDOWS_ACCEPTANCE.json"}
$root=Join-Path ([IO.Path]::GetTempPath()) ("orca-nova-acceptance-"+[guid]::NewGuid().ToString("N"))
$extract=Join-Path $root "portable"
$out=Join-Path $root "backend_stdout.log"
$errlog=Join-Path $root "backend_stderr.log"
$oldAppData=[Environment]::GetEnvironmentVariable("APPDATA","Process")
$oldTestMode=[Environment]::GetEnvironmentVariable("ORCA_NOVA_TEST_MODE","Process")
$appProcess=$null
try{
 New-Item -ItemType Directory -Force -Path $extract | Out-Null
 $zip=(Resolve-Path -LiteralPath $PortableZip).Path
 $shaText=Get-Content -LiteralPath (Resolve-Path -LiteralPath $ChecksumFile).Path -Raw
 $checksum=[regex]::Match($shaText,'(?im)^([a-f0-9]{64})\s+\*?ORCA_Nova_Portable_Windows\.zip\s*$')
 if(-not $checksum.Success){throw "Checksum manifest has no portable ZIP digest"}
 $actual=(Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash
 if($actual.ToLowerInvariant() -ne $checksum.Groups[1].Value.ToLowerInvariant()){throw "ZIP checksum mismatch"}
 $report.checksum_verified=$true
 Expand-Archive -LiteralPath $zip -DestinationPath $extract -Force
 $backend=Join-Path $extract "ORCA_Nova.exe"
 $native=Join-Path $extract "ORCA_Nova_UI.exe"
 if(-not(Test-Path -LiteralPath $backend -PathType Leaf)){throw "Backend EXE missing"}
 if(-not(Test-Path -LiteralPath $native -PathType Leaf)){throw "Native WebView2 EXE missing"}
 if(Get-ChildItem -LiteralPath $extract -File -Recurse | Where-Object {$_.Name -ieq "orca.exe"}){throw "Proprietary ORCA executable bundled"}
 $report.packaged_files_verified=$true
 $isolatedAppData=Join-Path $root "isolated-appdata"
 New-Item -ItemType Directory -Force -Path $isolatedAppData | Out-Null
 [Environment]::SetEnvironmentVariable("APPDATA",$isolatedAppData,"Process")
 [Environment]::SetEnvironmentVariable("ORCA_NOVA_TEST_MODE","1","Process")
 $appProcess=Start-Process -FilePath $backend -WorkingDirectory $extract -PassThru -RedirectStandardOutput $out -RedirectStandardError $errlog
 $url=""
 for($i=0;$i -lt 120;$i++){
  Start-Sleep -Milliseconds 250
  if(Test-Path -LiteralPath $out){
   $capture=Get-Content -LiteralPath $out -Raw -ErrorAction SilentlyContinue
   $match=[regex]::Match([string]$capture,'TEST_URL=(http://127\.0\.0\.1:\d+/\?session=[0-9a-f]+)')
   if($match.Success){$url=$match.Groups[1].Value;break}
  }
  if($appProcess.HasExited){break}
 }
 if(-not $url){throw "No TEST_URL from packaged Windows backend; GUI-subsystem stdout may be unavailable"}
 $report.backend_started=$true
 $parsed=[uri]$url
 $api=$parsed.GetLeftPart([uriPartial]::Authority)
 $token=([regex]::Match($url,'session=([0-9a-f]+)')).Groups[1].Value
 if(-not $token){throw "No API session token"}
 $sessionQuery="?session="+$token
 $auth=@{"X-ORCA-Session"=$token}
 $denied=$false
 try{Invoke-WebRequest -Uri ($api+"/api/status?session=invalid") -UseBasicParsing -TimeoutSec 8 | Out-Null}
 catch{
  $response=$_.Exception.Response
  if($null -ne $response -and [int]$response.StatusCode -eq 403){$denied=$true}
 }
 if(-not $denied){throw "Unauthenticated status unexpectedly accepted"}
 $report.session_restriction_verified=$true
 $status=Invoke-RestMethod -Method Get -Uri ($api+"/api/status"+$sessionQuery) -TimeoutSec 15
 if($status.platform -ne "windows" -or $status.version -ne $ExpectedVersion){throw ("Wrong platform/version "+$status.platform+"/"+$status.version)}
 $report.live_http_verified=$true
 $xyz=@'
3
Water
O 0.000000 0.000000 0.119262
H 0.000000 0.763239 -0.477047
H 0.000000 -0.763239 -0.477047
'@
 $inp=@'
! HF def2-SVP TightSCF
%pal
  nprocs 2
end
* xyzfile 0 1 water.xyz
'@
 $req=@{label="Public water smoke test";files=@(@{name="water.xyz";content=$xyz},@{name="water.inp";content=$inp});stages=@(@{name="H2O SP";input="water.inp";source="water.xyz";opt=$false})}
 $body=$req|ConvertTo-Json -Depth 8 -Compress
 $args=@{Method="Post";Uri=($api+"/api/preflight"+$sessionQuery);Headers=$auth;ContentType="application/json; charset=utf-8";Body=([Text.Encoding]::UTF8.GetBytes($body));TimeoutSec=15}
 $preflight=Invoke-RestMethod @args
 if($preflight.ok -ne $true){throw ("Valid water preflight failed: "+($preflight.errors -join "; "))}
 $report.preflight_verified=$true
 if($OrcaExe){
  if([IO.Path]::GetFileName($OrcaExe).ToLowerInvariant() -ne "orca.exe" -or -not(Test-Path -LiteralPath $OrcaExe -PathType Leaf)){throw "Provide a legally installed orca.exe full path"}
  $jobRoot=Join-Path $root "private-test-jobs"
  New-Item -ItemType Directory -Force -Path $jobRoot | Out-Null
  $cfg=@{orcaExe=(Resolve-Path -LiteralPath $OrcaExe).Path;workRoot=$jobRoot}|ConvertTo-Json -Compress
  $set=Invoke-RestMethod -Method Post -Uri ($api+"/api/config"+$sessionQuery) -Headers $auth -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes($cfg))
  if($set.ok -ne $true){throw "Failed to configure isolated ORCA"}
  $run=Invoke-RestMethod -Method Post -Uri ($api+"/api/run"+$sessionQuery) -Headers $auth -ContentType "application/json; charset=utf-8" -Body ([Text.Encoding]::UTF8.GetBytes($body))
  if($run.ok -ne $true){throw "Real ORCA job rejected"}
  $finish=""
  $last=$null
  $until=(Get-Date).AddSeconds($RealOrcaTimeoutSeconds)
  while((Get-Date) -lt $until){
   Start-Sleep -Seconds 2
   $last=Invoke-RestMethod -Method Get -Uri ($api+"/api/status"+$sessionQuery) -TimeoutSec 15
   if($null -ne $last.job -and $last.job.state -in @("completed","failed","interrupted")){$finish=$last.job.state;break}
  }
  if(-not $finish){
   Invoke-RestMethod -Method Post -Uri ($api+"/api/stop"+$sessionQuery) -Headers $auth -ContentType "application/json" -Body "{}" | Out-Null
   throw "Real ORCA timed out; stop requested"
  }
  if($finish -ne "completed"){throw ("Real ORCA status "+$finish+": "+$last.job.message)}
  $outputFile=Join-Path $run.folder "water.out"
  if(-not(Test-Path -LiteralPath $outputFile -PathType Leaf)){throw "Completed ORCA calculation has no water.out"}
  $outputText=Get-Content -LiteralPath $outputFile -Raw
  if($outputText -notmatch "ORCA TERMINATED NORMALLY" -or $outputText -notmatch "FINAL SINGLE POINT ENERGY"){throw "Real ORCA missing termination or energy"}
  $report.real_orca_execution="PASS"
 }
 $report.passed=$true
 Write-Host "PASS: Windows Portable API smoke; Real ORCA: $($report.real_orca_execution)"
}catch{
 $report.errors=@([string]$_.Exception.Message)
 Write-Warning ("ORCA Nova acceptance FAILED: "+$_.Exception.Message)
}finally{
 if($null -ne $appProcess -and -not $appProcess.HasExited){& taskkill.exe /PID $appProcess.Id /T /F 2>$null | Out-Null}
 [Environment]::SetEnvironmentVariable("APPDATA",$oldAppData,"Process")
 [Environment]::SetEnvironmentVariable("ORCA_NOVA_TEST_MODE",$oldTestMode,"Process")
 $parent=Split-Path -Parent $ReportPath
 if($parent -and -not(Test-Path -LiteralPath $parent)){New-Item -ItemType Directory -Force -Path $parent|Out-Null}
 $report|ConvertTo-Json -Depth 8|Set-Content -LiteralPath $ReportPath -Encoding UTF8
 if(-not $KeepTemporaryFiles -and (Test-Path -LiteralPath $root)){Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue}
}
if(-not $report.passed){exit 1}
