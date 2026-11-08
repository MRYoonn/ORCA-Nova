//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// The Edge --app host has its own window and may show a generic globe.
// This best-effort patch sets WM_SETICON on the actual app window. Native
// WebView2 hosting is the reliable future route for pinned taskbar identity.
func patchWindowsAppIcon() {
	iconBytes, e := assets.ReadFile("assets/ORCA_Nova.ico")
	if e != nil {
		return
	}
	base, e := os.UserCacheDir()
	if e != nil {
		return
	}
	dir := filepath.Join(base, "ORCA Nova")
	if os.MkdirAll(dir, 0700) != nil {
		return
	}
	p := filepath.Join(dir, "orca-taskbar.ico")
	if os.WriteFile(p, iconBytes, 0600) != nil {
		return
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	loadImage := user32.NewProc("LoadImageW")
	sendMessage := user32.NewProc("SendMessageW")
	enumWindows := user32.NewProc("EnumWindows")
	getText := user32.NewProc("GetWindowTextW")
	getTextLen := user32.NewProc("GetWindowTextLengthW")
	setClass := user32.NewProc("SetClassLongPtrW")
	ptr, _ := syscall.UTF16PtrFromString(p)
	hicon, _, _ := loadImage.Call(0, uintptr(unsafe.Pointer(ptr)), 1, 0, 0, 0x10|0x40)
	if hicon == 0 {
		return
	}
	start := time.Now()
	for time.Since(start) < 22*time.Second {
		found := false
		cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
			n, _, _ := getTextLen.Call(hwnd)
			if n == 0 || n > 240 {
				return 1
			}
			title := make([]uint16, int(n)+1)
			getText.Call(hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
			t := syscall.UTF16ToString(title)
			if strings.Contains(t, "ORCA Nova") {
				// Do not change unrelated windows that happen to have the same text.
				// This only changes the icon display; Windows may still group by Edge AppID.
				sendMessage.Call(hwnd, 0x80, 1, hicon)
				sendMessage.Call(hwnd, 0x80, 0, hicon)
				setClass.Call(hwnd, ^uintptr(13), hicon) // GCLP_HICON=-14
				setClass.Call(hwnd, ^uintptr(33), hicon) // GCLP_HICONSM=-34
				found = true
			}
			return 1
		})
		enumWindows.Call(cb, 0)
		if found {
			return
		}
		time.Sleep(800 * time.Millisecond)
	}
}
