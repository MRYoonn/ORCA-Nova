package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type updaterTransport func(*http.Request) (*http.Response, error)

func (f updaterTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func updateResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Status: fmt.Sprint(code), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func TestUpdaterReleaseCheckClearsStaleAndFiltersPrerelease(t *testing.T) {
	u := newUpdater()
	rel := githubRelease{Tag: "v99.0.0", HTMLURL: "https://github.com/" + updateRepository + "/releases/tag/v99.0.0", Assets: []updateAsset{{Name: releaseZipName}, {Name: "SHA256SUMS.txt"}}}
	u.client.Transport = updaterTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" || r.Header.Get("User-Agent") == "" {
			t.Fatal("release check provenance")
		}
		b, _ := json.Marshal(rel)
		return updateResponse(200, string(b)), nil
	})
	if !u.check().Available {
		t.Fatal("valid release missing")
	}
	rel.Prerelease = true
	if u.check().Available {
		t.Fatal("prerelease offered")
	}
	rel.Prerelease = false
	rel.Draft = true
	if u.check().Available {
		t.Fatal("draft offered")
	}
	rel.Draft = false
	rel.Assets = nil
	if u.check().Available {
		t.Fatal("incomplete portable release offered")
	}
	u.client.Transport = updaterTransport(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("offline") })
	u.check()
	state := u.snapshot()
	if state.Available || state.Checking || state.Latest != "" || u.release.Tag != "" {
		t.Fatal("offline check retained stale installable release", state)
	}
}
func TestUpdaterDownloadLimitStatusAndURLTrust(t *testing.T) {
	u := newUpdater()
	calls := 0
	u.client.Transport = updaterTransport(func(*http.Request) (*http.Response, error) { calls++; return updateResponse(200, "12345"), nil })
	good := "https://github.com/" + updateRepository + "/releases/download/v99/asset"
	if b, err := u.getFile(good, 5); err != nil || string(b) != "12345" {
		t.Fatal(string(b), err)
	}
	if _, err := u.getFile(good, 4); err == nil {
		t.Fatal("oversized download accepted")
	}
	before := calls
	for _, url := range []string{"http://github.com/" + updateRepository + "/releases/download/v99/a", "https://evil.test/a", "https://github.com/other/repo/releases/download/v99/a"} {
		if _, err := u.getFile(url, 100); err == nil {
			t.Fatal("untrusted URL accepted", url)
		}
	}
	if calls != before {
		t.Fatal("untrusted URL reached transport")
	}
	u.client.Transport = updaterTransport(func(*http.Request) (*http.Response, error) { return updateResponse(403, "denied"), nil })
	if _, err := u.getFile(good, 100); err == nil {
		t.Fatal("403 treated as archive")
	}
}
func TestUpdaterChecksumBeforePayloadUse(t *testing.T) {
	data := []byte("portable payload")
	digest := sha256.Sum256(data)
	if err := verifyReleaseDigest(data, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	if verifyReleaseDigest(append(data, 'x'), hex.EncodeToString(digest[:])) == nil {
		t.Fatal("corrupted artifact accepted")
	}
}
func mockPortableZip(t *testing.T, names ...string) string {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range names {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte("mock payload"))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "release.zip")
	if err := os.WriteFile(p, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestUpdaterArchiveValidationAndExistingSettingsPreserved(t *testing.T) {
	stage := t.TempDir()
	settings := filepath.Join(stage, "config.json")
	os.WriteFile(settings, []byte("existing preferences"), 0600)
	p := mockPortableZip(t, "ORCA_Nova.exe", "ORCA_Nova_UI.exe", "runtimes/win-x64/native/WebView2Loader.dll")
	if err := stageReleaseArchive(p, stage); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(settings)
	if string(b) != "existing preferences" {
		t.Fatal("settings overwritten")
	}
	for _, name := range []string{"../outside.dll", "C:/outside.dll", "config.json", "launch.ps1", "unexpected.exe", "/absolute.dll"} {
		t.Run(name, func(t *testing.T) {
			if stageReleaseArchive(mockPortableZip(t, "ORCA_Nova.exe", name), t.TempDir()) == nil {
				t.Fatal("unsafe payload accepted", name)
			}
		})
	}
	if stageReleaseArchive(mockPortableZip(t, "only.dll"), t.TempDir()) == nil {
		t.Fatal("missing backend accepted")
	}
	if stageReleaseArchive(mockPortableZip(t, "ORCA_Nova.exe", "ORCA_Nova.exe"), t.TempDir()) == nil {
		t.Fatal("duplicate payload path accepted")
	}
}
func TestUpdaterRedirectTrust(t *testing.T) {
	for _, destination := range []string{"http://github.com/file", "https://evil.test/file"} {
		t.Run(destination, func(t *testing.T) {
			u := newUpdater()
			u.client.Transport = updaterTransport(func(*http.Request) (*http.Response, error) {
				r := updateResponse(302, "")
				r.Header.Set("Location", destination)
				return r, nil
			})
			if _, err := u.getFile("https://github.com/"+updateRepository+"/releases/download/v99/a", 100); err == nil {
				t.Fatal("unsafe redirect accepted")
			}
		})
	}
}
func TestUpdaterNestedBackendDoesNotSatisfyRootExecutable(t *testing.T) {
	if stageReleaseArchive(mockPortableZip(t, "subdir/ORCA_Nova.exe"), t.TempDir()) == nil {
		t.Fatal("nested backend accepted without root executable")
	}
}
