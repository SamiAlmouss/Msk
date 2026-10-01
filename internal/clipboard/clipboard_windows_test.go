//go:build windows

package clipboard

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

// Clipboard belongs to a window station. A private station and desktop let us
// exercise the real Windows adapter without reading or altering the user's clipboard.
func TestNativeClipboardInPrivateStation(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	getStation := user.NewProc("GetProcessWindowStation")
	setStation := user.NewProc("SetProcessWindowStation")
	createStation := user.NewProc("CreateWindowStationW")
	closeStation := user.NewProc("CloseWindowStation")
	createDesktop := user.NewProc("CreateDesktopW")
	closeDesktop := user.NewProc("CloseDesktop")
	getDesktop := user.NewProc("GetThreadDesktop")
	setDesktop := user.NewProc("SetThreadDesktop")
	tid, _, _ := kernel.NewProc("GetCurrentThreadId").Call()
	originalDesktop, _, _ := getDesktop.Call(tid)
	originalStation, _, _ := getStation.Call()
	name, _ := syscall.UTF16PtrFromString(fmt.Sprintf("msk-test-%d", os.Getpid()))
	station, _, e := createStation.Call(uintptr(unsafe.Pointer(name)), 0, 0x37f, 0)
	if station == 0 {
		t.Skipf("private window station unavailable: %v", e)
	}
	defer closeStation.Call(station)
	ok, _, e := setStation.Call(station)
	if ok == 0 {
		t.Skipf("private station access unavailable: %v", e)
	}
	defer setStation.Call(originalStation)
	desktopName, _ := syscall.UTF16PtrFromString("msk-test-desktop")
	desktop, _, e := createDesktop.Call(uintptr(unsafe.Pointer(desktopName)), 0, 0, 0, 0x1ff, 0)
	if desktop == 0 {
		t.Skipf("private desktop unavailable: %v", e)
	}
	defer closeDesktop.Call(desktop)
	ok, _, e = setDesktop.Call(desktop)
	if ok == 0 {
		t.Skipf("private desktop thread access unavailable: %v", e)
	}
	defer setDesktop.Call(originalDesktop)
	c := System{}
	if _, e := c.Read(); e == nil {
		t.Fatal("empty private Clipboard was accepted")
	}
	expected := "مشروع/\r\n└── ملف.txt\r\nUnicode 😀"
	if e := c.Write(expected); e != nil {
		t.Fatal(e)
	}
	actual, e := c.Read()
	if e != nil {
		t.Fatal(e)
	}
	if actual != expected {
		t.Fatalf("Unicode round trip failed: %q", actual)
	}
	if e := c.Write("updated"); e != nil {
		t.Fatal(e)
	}
	actual, e = c.Read()
	if e != nil || actual != "updated" {
		t.Fatal("replacement failed", e)
	}
}
