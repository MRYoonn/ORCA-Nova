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
	return &updater{state: updateState{Current: appVersion, Message: "업데이트 확인 전"}, client: &http.Client{Timeout: 15 * time.Second}}
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
func newerVersion(a, b string) bool {
	va, ok := parseVersion(a)
	vb, ok2 := parseVersion(b)
	if !ok || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if va[i] > vb[i] {
			return true
		}
		if va[i] < vb[i] {
			return false
		}
	}
	// A final release supersedes its same-core release candidate.
	_, _, candidateHasPre := strings.Cut(strings.TrimSpace(a), "-")
	_, currentPre, currentHasPre := strings.Cut(strings.TrimSpace(b), "-")
	return !candidateHasPre && currentHasPre && currentPre != ""
}
func (u *updater) check() updateState {
	u.mu.Lock()
	if u.state.Checking {
		res := u.state
		u.mu.Unlock()
		return res
	}
	u.state.Checking = true
	// Failed/offline checks must not offer stale cached updates.
	u.state.Available = false
	u.state.Latest = ""
	u.state.ReleaseURL = ""
	u.release = githubRelease{}
	u.state.Message = "GitHub Release 확인 중"
	u.mu.Unlock()
	defer func() {
		u.mu.Lock()
		u.state.Checking = false
		u.state.LastChecked = time.Now().Format(time.RFC3339)
		u.mu.Unlock()
	}()
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+updateRepository+"/releases/latest", nil)
	req.Header.Set("User-Agent", "ORCA-Nova/"+appVersion)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.client.Do(req)
	if err != nil {
		u.mu.Lock()
		u.state.Message = "업데이트 서버 접속 실패: " + err.Error()
		s := u.state
		u.mu.Unlock()
		return s
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		u.mu.Lock()
		if resp.StatusCode == 404 {
			u.state.Message = "GitHub Release가 아직 없거나 비공개 저장소예요."
		} else {
			u.state.Message = "GitHub 응답: " + resp.Status
		}
		s := u.state
		u.mu.Unlock()
		return s
	}
	var rel githubRelease
	if err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		u.mu.Lock()
		u.state.Message = "Release 정보 해석 실패"
		s := u.state
		u.mu.Unlock()
		return s
	}
	available := !rel.Draft && !rel.Prerelease && newerVersion(rel.Tag, appVersion)
	foundZip, foundSum := false, false
	for _, a := range rel.Assets {
		if a.Name == releaseZipName {
			foundZip = true
		}
		if a.Name == "SHA256SUMS.txt" {
			foundSum = true
		}
	}
	if !foundZip || !foundSum {
		available = false
	}
	u.mu.Lock()
	u.release = rel
	u.state.Latest = rel.Tag
	u.state.Available = available
	u.state.ReleaseURL = rel.HTMLURL
	if !foundZip || !foundSum {
		u.state.Message = "Release에 Portable ZIP 또는 SHA256SUMS.txt가 없어요."
	} else if available {
		u.state.Message = "새 버전 " + rel.Tag + " 사용 가능"
	} else {
		u.state.Message = "최신 버전 사용 중"
	}
	s := u.state
	u.mu.Unlock()
	return s
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
	if err = os.WriteFile(script, []byte(updateScript), 0600); err != nil {
		return "", err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script, "-CurrentPid", strconv.Itoa(os.Getpid()), "-StageDir", stageDir, "-TargetDir", filepath.Dir(dst), "-Launch", dst, "-Log", filepath.Join(tmpDir, "UPDATE_LOG.txt"))
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return tmpDir, nil
}

const updateScript = `param([int]$CurrentPid,[string]$StageDir,[string]$TargetDir,[string]$Launch,[string]$Log)
$ErrorActionPreference='Stop'
try {
  for($i=0;$i -lt 90;$i++) { $p=Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue; if(-not $p){break};Start-Sleep -Seconds 1 }
  if (Get-Process -Id $CurrentPid -ErrorAction SilentlyContinue){throw '기존 ORCA Nova가 닫히지 않았습니다.'}
  $appExe = Join-Path $TargetDir 'ORCA_Nova.exe'
  $backup = $appExe + '.previous'
  if(Test-Path -LiteralPath $backup){Remove-Item -LiteralPath $backup -Force}
  Copy-Item -LiteralPath $appExe -Destination $backup -Force
  # The WebView2 native host may still be exiting. Retry boundedly if its files are locked.
  $copied=$false
  for($i=0;$i -lt 45;$i++) {
    try {
      Copy-Item -Path (Join-Path $StageDir '*') -Destination $TargetDir -Recurse -Force -ErrorAction Stop
      $copied=$true;break
    } catch {
      Start-Sleep -Seconds 2
    }
  }
  if(-not $copied) {Copy-Item -LiteralPath $backup -Destination $appExe -Force;throw '업데이트 파일 교체 실패. EXE를 복구했습니다.'}
  Start-Process -FilePath $Launch -WorkingDirectory $TargetDir
  ('Updated: '+(Get-Date)) | Set-Content -LiteralPath $Log -Encoding UTF8
}catch{('ERROR: '+$_.Exception.Message) | Set-Content -LiteralPath $Log -Encoding UTF8}
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
