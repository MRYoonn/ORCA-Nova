package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var appVersion = "1.5.0"

const updateRepository = "MRYoonn/ORCA-Nova"
const releaseZipName = "ORCA_Nova_Portable_Windows.zip"

type updateAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}
type githubRelease struct {
	Tag        string        `json:"tag_name"`
	HTMLURL    string        `json:"html_url"`
	Assets     []updateAsset `json:"assets"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
}
type updateState struct {
	Current     string `json:"current"`
	Channel     string `json:"channel"`
	Prerelease  bool   `json:"prerelease"`
	Latest      string `json:"latest,omitempty"`
	Available   bool   `json:"available"`
	Checking    bool   `json:"checking"`
	Message     string `json:"message"`
	ReleaseURL  string `json:"releaseUrl,omitempty"`
	LastChecked string `json:"lastChecked,omitempty"`
}
type updater struct {
	mu      sync.Mutex
	state   updateState
	release githubRelease
	client  *http.Client
}

func newUpdater() *updater {
	return &updater{state: updateState{Current: appVersion, Channel: "stable", Message: "업데이트 확인 전"}, client: &http.Client{Timeout: 15 * time.Second}}
}
func (u *updater) snapshot() updateState { u.mu.Lock(); defer u.mu.Unlock(); return u.state }
func parseVersion(v string) ([3]int, bool) {
	var arr [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return arr, false
	}
	for i, p := range parts {
		n, e := strconv.Atoi(p)
		if e != nil || n < 0 {
			return arr, false
		}
		arr[i] = n
	}
	return arr, true
}
func releaseVersionSuffix(v string) (string, int, bool) {
 _, suffix, prerelease := strings.Cut(strings.TrimPrefix(strings.TrimSpace(v), "v"), "-")
 if !prerelease {return "",0,true}
 if !strings.HasPrefix(suffix,"rc.") {return "",0,false}
 n,err:=strconv.Atoi(strings.TrimPrefix(suffix,"rc."))
 if err!=nil||n<1{return "",0,false}
 return "rc",n,true
}
func newerVersion(a,b string)bool {
 va,ok:=parseVersion(a);vb,ok2:=parseVersion(b)
 if !ok||!ok2{return false}
 ak,an,validA:=releaseVersionSuffix(a)
 bk,bn,validB:=releaseVersionSuffix(b)
 if !validA||!validB{return false}
 for i:=0;i<3;i++{if va[i]>vb[i]{return true};if va[i]<vb[i]{return false}}
 if ak==bk{if ak=="rc"{return an>bn};return false}
 return ak==""&&bk=="rc"
}
func releaseAssetsValid(rel githubRelease)bool {
 zip,sum:=false,false
 for _,asset:=range rel.Assets{
  if asset.Name==releaseZipName{zip=true}
  if asset.Name=="SHA256SUMS.txt"{sum=true}
 }
 return zip&&sum
}
func releaseFitsChannel(rel githubRelease,channel string)bool {
 if rel.Draft||!releaseAssetsValid(rel){return false}
 if _,ok:=parseVersion(rel.Tag);!ok{return false}
 kind,_,ok:=releaseVersionSuffix(rel.Tag);if !ok{return false}
 if rel.Prerelease!=(kind=="rc"){return false}
 return channel=="beta"||!rel.Prerelease
}
func selectRelease(candidates []githubRelease,channel,current string)(githubRelease,bool){
 var best githubRelease
 found:=false
 for _,candidate:=range candidates {
  if !releaseFitsChannel(candidate,channel)||!newerVersion(candidate.Tag,current){continue}
  if !found||newerVersion(candidate.Tag,best.Tag){best=candidate;found=true}
 }
 return best,found
}
func (u *updater) setChannel(channel string)(updateState,error){
 if channel!="stable"&&channel!="beta"{return u.snapshot(),fmt.Errorf("업데이트 채널은 Stable/Beta만 지원합니다")}
 u.mu.Lock()
 defer u.mu.Unlock()
 if u.state.Checking {return u.state,fmt.Errorf("업데이트 검색 중에는 채널을 변경할 수 없습니다")}
 if u.state.Channel==channel{return u.state,nil}
 u.state.Channel=channel
 u.state.Latest=""
 u.state.Available=false
 u.state.Prerelease=false
 u.state.ReleaseURL=""
 u.state.LastChecked=""
 u.release=githubRelease{}
 u.state.Message=map[string]string{"stable":"Stable 채널 · 정식판만 확인합니다","beta":"Beta 채널 · 시험판을 포함합니다"}[channel]
 return u.state,nil
}
func (u *updater) check() updateState {
 u.mu.Lock()
 if u.state.Checking{res:=u.state;u.mu.Unlock();return res}
 channel:=u.state.Channel
 if channel!="beta"{channel="stable";u.state.Channel="stable"}
 u.state.Checking=true
 u.state.Available=false
 u.state.Prerelease=false
 u.state.Latest=""
 u.state.ReleaseURL=""
 u.release=githubRelease{}
 u.state.Message="GitHub "+strings.ToUpper(channel[:1])+channel[1:]+" Release 확인 중"
 u.mu.Unlock()
 defer func(){
  u.mu.Lock()
  u.state.Checking=false
  u.state.LastChecked=time.Now().Format(time.RFC3339)
  u.mu.Unlock()
 }()
 endpoint:="https://api.github.com/repos/"+updateRepository+"/releases/latest"
 if channel=="beta"{endpoint="https://api.github.com/repos/"+updateRepository+"/releases?per_page=50"}
 req,_:=http.NewRequest(http.MethodGet,endpoint,nil)
 req.Header.Set("User-Agent","ORCA-Nova/"+appVersion)
 req.Header.Set("Accept","application/vnd.github+json")
 resp,err:=u.client.Do(req)
 if err!=nil{
  u.mu.Lock();u.state.Message="업데이트 서버 접속 실패: "+err.Error();out:=u.state;u.mu.Unlock();return out
 }
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK {
  u.mu.Lock()
  if resp.StatusCode==404{u.state.Message="GitHub Release가 아직 없거나 비공개 저장소예요."}else{u.state.Message="GitHub 응답: "+resp.Status}
  out:=u.state;u.mu.Unlock();return out
 }
 var releases []githubRelease
 if channel=="beta"{
  if err=json.NewDecoder(io.LimitReader(resp.Body,1<<20)).Decode(&releases);err!=nil{
   u.mu.Lock();u.state.Message="Release 목록 해석 실패";out:=u.state;u.mu.Unlock();return out
  }
 }else{
  var rel githubRelease
  if err=json.NewDecoder(io.LimitReader(resp.Body,1<<20)).Decode(&rel);err!=nil{
   u.mu.Lock();u.state.Message="Release 정보 해석 실패";out:=u.state;u.mu.Unlock();return out
  }
  releases=[]githubRelease{rel}
 }
 rel,available:=selectRelease(releases,channel,appVersion)
 u.mu.Lock()
 if available {
  u.release=rel
  u.state.Latest=rel.Tag
  u.state.Available=true
  u.state.Prerelease=rel.Prerelease
  u.state.ReleaseURL=rel.HTMLURL
  u.state.Message=rel.Tag+" 사용 가능 · 릴리스 노트 확인 후 승인해 주세요"
 }else{
  u.state.Message=map[string]string{"stable":"Stable 최신 버전 사용 중 (Beta 제외)","beta":"Beta 채널에서 더 새로운 버전이 없습니다"}[channel]
 }
 out:=u.state
 u.mu.Unlock()
 return out
}
func (u *updater) getFile(url string, max int64) ([]byte, error) {
	if !strings.HasPrefix(url, "https://github.com/"+updateRepository+"/releases/download/") && !strings.HasPrefix(url, "https://objects.githubusercontent.com/") {
		return nil, errors.New("신뢰할 수 없는 GitHub Release 다운로드 URL")
	}
	client := &http.Client{Timeout: 4 * time.Minute, Transport: u.client.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 {
			return errors.New("redirect excess")
		}
		if req.URL.Scheme != "https" {
			return errors.New("HTTPS only")
		}
		host := strings.ToLower(req.URL.Hostname())
		if host != "github.com" && host != "objects.githubusercontent.com" && host != "release-assets.githubusercontent.com" {
			return errors.New("untrusted redirect host: " + host)
		}
		return nil
	}}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "ORCA-Nova/"+appVersion)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download response %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("download too large")
	}
	return b, nil
}
func checksumFor(sumText, filename string) (string, error) {
	scan := bufio.NewScanner(strings.NewReader(sumText))
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) >= 2 && strings.TrimLeft(fields[1], "*") == filename {
			s := strings.ToLower(fields[0])
			if len(s) != 64 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(s) {
				return "", errors.New("checksum invalid")
			}
			return s, nil
		}
	}
	return "", errors.New("release checksum entry not found")
}
func (u *updater) prepare() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("Windows에서만 업데이트 설치가 가능해요")
	}
	u.mu.Lock()
	rel := u.release
	available := u.state.Available
	u.mu.Unlock()
	if !available {
		return "", errors.New("설치할 새 버전이 없어요. 업데이트 확인을 먼저 실행하세요")
	}
	var zipURL, shaURL string
	for _, a := range rel.Assets {
		if a.Name == releaseZipName {
			zipURL = a.URL
		}
		if a.Name == "SHA256SUMS.txt" {
			shaURL = a.URL
		}
	}
	if zipURL == "" || shaURL == "" {
		return "", errors.New("release files missing")
	}
	shaData, err := u.getFile(shaURL, 512*1024)
	if err != nil {
		return "", err
	}
	want, err := checksumFor(string(shaData), releaseZipName)
	if err != nil {
		return "", err
	}
	zipData, err := u.getFile(zipURL, 300*1024*1024)
	if err != nil {
		return "", err
	}
	if err = verifyReleaseDigest(zipData, want); err != nil {
		return "", err
	}
	dst, err := os.Executable()
	if err != nil {
		return "", err
	}
	if strings.ToLower(filepath.Base(dst)) != "orca_nova.exe" {
		return "", errors.New("실행 파일명이 ORCA_Nova.exe여야 자동 교체할 수 있어요")
	}
	// Verify SHA-256 before extracting. Stage the entire portable payload, including native UI and WebView2 DLLs.
	tmpDir, err := os.MkdirTemp("", "orca-nova-update-")
	if err != nil {
		return "", err
	}
	zipPath := filepath.Join(tmpDir, "release.zip")
	if err = os.WriteFile(zipPath, zipData, 0600); err != nil {
		return "", err
	}
	stageDir := filepath.Join(tmpDir, "stage")
	if err = stageReleaseArchive(zipPath, stageDir); err != nil {
		return "", err
	}
	script := filepath.Join(tmpDir, "apply_update.ps1")
	if err = os.WriteFile(script, powershellScriptBytes(updateScript), 0600); err != nil {
		return "", err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-CurrentPid", strconv.Itoa(os.Getpid()), "-StageDir", stageDir, "-TargetDir", filepath.Dir(dst), "-Launch", dst, "-Log", filepath.Join(tmpDir, "UPDATE_LOG.txt"))
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return tmpDir, nil
}

// Windows PowerShell 5.1 assumes ANSI for .ps1 files without a BOM.
// All update scripts must be written as UTF-8 with BOM to preserve Korean
// diagnostics and avoid malformed PowerShell string literals.
func powershellScriptBytes(s string) []byte {
    return append([]byte{0xef, 0xbb, 0xbf}, []byte(s)...)
}

const updateScript = `param([int]$CurrentPid,[string]$StageDir,[string]$TargetDir,[string]$Launch,[string]$Log)
$ErrorActionPreference='Stop'
$entries=@()
$replacing=$false
$backupDir=Join-Path $TargetDir ('.ORCA_Nova_backup_'+(Get-Date -Format 'yyyyMMddHHmmssfff'))
$errorsDuringRollback=@()
try {
  for($i=0;$i -lt 90;$i++){
    $p=Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue
    if(-not $p){break}
    Start-Sleep -Seconds 1
  }
  if(Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue){
    throw '기존 ORCA Nova가 종료되지 않아 업데이트를 중단했습니다.'
  }
  $appExe=Join-Path $TargetDir 'ORCA_Nova.exe'
  if(-not (Test-Path -LiteralPath $appExe -PathType Leaf)){throw '기존 ORCA_Nova.exe가 없습니다.'}
  if(-not (Test-Path -LiteralPath (Join-Path $StageDir 'ORCA_Nova.exe') -PathType Leaf)){
    throw '업데이트 ZIP의 ORCA_Nova.exe가 없습니다.'
  }
  $root=[System.IO.Path]::GetFullPath($StageDir).TrimEnd([char]92,[char]47)+[System.IO.Path]::DirectorySeparatorChar
  $stageFiles=@(Get-ChildItem -LiteralPath $StageDir -File -Recurse -ErrorAction Stop)
  if($stageFiles.Count -lt 1){throw '새 버전 파일이 없습니다.'}
  # Back up ALL files that the update may replace, not just the backend EXE.
  # Any non-overlapping user file in the target folder remains untouched.
  foreach($f in $stageFiles){
    $relative=$f.FullName.Substring($root.Length)
    if($relative -eq '' -or $relative -match '(^|[\\/])\.\.([\\/]|$)'){
      throw '잘못된 업데이트 대상 경로입니다.'
    }
    $destination=Join-Path $TargetDir $relative
    $backup=Join-Path $backupDir $relative
    if(Test-Path -LiteralPath $destination -PathType Container){
      throw ('기존 폴더와 설치 파일이 충돌합니다: '+$relative)
    }
    $existed=Test-Path -LiteralPath $destination -PathType Leaf
    if($existed){
      New-Item -ItemType Directory -Force -Path (Split-Path -Parent $backup) | Out-Null
      Copy-Item -LiteralPath $destination -Destination $backup -Force -ErrorAction Stop
    }
    $entries+= [pscustomobject]@{Src=$f.FullName;Dst=$destination;Backup=$backup;Exists=$existed;Name=$relative}
  }
  $replacing=$true
  foreach($item in $entries){
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $item.Dst) | Out-Null
    $copied=$false
    for($i=0;$i -lt 45;$i++){
      try{
        Copy-Item -LiteralPath $item.Src -Destination $item.Dst -Force -ErrorAction Stop
        $copied=$true
        break
      }catch{
        if($i -lt 44){Start-Sleep -Seconds 2}
      }
    }
    if(-not $copied){throw ('업데이트 파일 교체 실패: '+$item.Name)}
  }
  Start-Process -FilePath $Launch -WorkingDirectory $TargetDir -ErrorAction Stop
  ('SUCCESS: '+(Get-Date)+'; backup: '+$backupDir) | Set-Content -LiteralPath $Log -Encoding UTF8
  exit 0
}catch{
  $cause=$_.Exception.Message
  if($replacing){
    foreach($item in $entries){
      try{
        if($item.Exists){
          if(-not(Test-Path -LiteralPath $item.Backup -PathType Leaf)){
            throw ('백업 누락: '+$item.Name)
          }
          $restored=$false
          for($i=0;$i -lt 20;$i++){
            try{
              Copy-Item -LiteralPath $item.Backup -Destination $item.Dst -Force -ErrorAction Stop
              $restored=$true
              break
            }catch{
              if($i -lt 19){Start-Sleep -Seconds 1}
            }
          }
          if(-not $restored){throw ('복구 실패: '+$item.Name)}
        }elseif(Test-Path -LiteralPath $item.Dst){
          Remove-Item -LiteralPath $item.Dst -Force -ErrorAction Stop
        }
      }catch{$errorsDuringRollback+=($_.Exception.Message)}
    }
  }
  $message='ERROR: '+$cause
  if($errorsDuringRollback.Count -eq 0){
    $message+='; rollback: '+$(if($replacing){'SUCCESS'}else{'NOT NEEDED'})
  }else{
    $message+='; rollback: INCOMPLETE; '+($errorsDuringRollback -join '; ')
  }
  $message | Set-Content -LiteralPath $Log -Encoding UTF8
  exit 1
}
`

// stageReleaseArchive validates portable payload paths and types before installation.
func stageReleaseArchive(zipPath, stageDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	if err = os.MkdirAll(stageDir, 0700); err != nil {
		return err
	}
	foundExe := false
	var total uint64
	for _, z := range zr.File {
		if strings.HasPrefix(z.Name, "/") || strings.HasPrefix(z.Name, `\`) {
			return errors.New("rooted ZIP entry path")
		}
		name := filepath.Clean(filepath.FromSlash(z.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || strings.Contains(name, ":") {
			return errors.New("invalid ZIP entry path")
		}
		if z.FileInfo().IsDir() {
			continue
		}
		// No scripts, user files, or arbitrary executables are allowed inside the auto-update payload.
		lower := strings.ToLower(filepath.Base(name))
		ext := strings.ToLower(filepath.Ext(name))
		if !(lower == "orca_nova.exe" || lower == "orca_nova_ui.exe" || lower == "orca_nova.ico" || ext == ".dll" || ext == ".config") {
			return errors.New("unexpected release file: " + name)
		}
		if strings.EqualFold(name, "ORCA_Nova.exe") {
			foundExe = true
		}
		if z.UncompressedSize64 > 100*1024*1024 {
			return errors.New("release file too large")
		}
		total += z.UncompressedSize64
		if total > 350*1024*1024 {
			return errors.New("release payload too large")
		}
		target := filepath.Join(stageDir, name)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		f, err := z.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			f.Close()
			return err
		}
		n, copyErr := io.Copy(out, io.LimitReader(f, 100*1024*1024+1))
		closeErr := out.Close()
		f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n > 100*1024*1024 {
			return errors.New("release file too large")
		}
	}
	if !foundExe {
		return errors.New("release ZIP missing ORCA_Nova.exe")
	}

	return nil
}

func verifyReleaseDigest(data []byte, want string) error {
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return errors.New("SHA-256 검사 실패: 다운로드를 중단했어요")
	}
	return nil
}
