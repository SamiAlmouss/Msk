//go:build windows

package clipboard

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var user = syscall.NewLazyDLL("user32.dll")
var kernel = syscall.NewLazyDLL("kernel32.dll")
var open = user.NewProc("OpenClipboard")
var closeClip = user.NewProc("CloseClipboard")
var get = user.NewProc("GetClipboardData")
var set = user.NewProc("SetClipboardData")
var empty = user.NewProc("EmptyClipboard")
var createWindow = user.NewProc("CreateWindowExW")
var destroyWindow = user.NewProc("DestroyWindow")
var lock = kernel.NewProc("GlobalLock")
var unlock = kernel.NewProc("GlobalUnlock")
var alloc = kernel.NewProc("GlobalAlloc")
var free = kernel.NewProc("GlobalFree")
var size = kernel.NewProc("GlobalSize")
var moveMemory = kernel.NewProc("RtlMoveMemory")

// A hidden owner window is required for reliable SetClipboardData after EmptyClipboard.
func session(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := syscall.UTF16PtrFromString("STATIC")
	hwnd, _, e := createWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		return fmt.Errorf("create Clipboard owner: %v", e)
	}
	defer destroyWindow.Call(hwnd)
	for i := 0; i < 10; i++ {
		ok, _, err := open.Call(hwnd)
		if ok != 0 {
			defer closeClip.Call()
			return fn()
		}
		e = err
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("Clipboard is busy or unavailable: %v", e)
}
func (System) Read() (text string, err error) {
	err = session(func() error {
		h, _, e := get.Call(13)
		if h == 0 {
			return fmt.Errorf("Clipboard has no Unicode text: %v", e)
		}
		bytes, _, _ := size.Call(h)
		if bytes < 2 || bytes%2 != 0 || bytes > 32<<20 {
			return fmt.Errorf("Clipboard text is empty or exceeds 16 MiB UTF-16 limit")
		}
		p, _, e := lock.Call(h)
		if p == 0 {
			return fmt.Errorf("lock Clipboard: %v", e)
		}
		defer unlock.Call(h)
		data := make([]uint16, int(bytes/2))
		moveMemory.Call(uintptr(unsafe.Pointer(&data[0])), p, uintptr(len(data)*2))
		runtime.KeepAlive(data)
		end := 0
		for end < len(data) && data[end] != 0 {
			end++
		}
		if end == len(data) {
			return fmt.Errorf("Clipboard text is not NUL terminated")
		}
		text = string(utf16.Decode(data[:end]))
		if text == "" {
			return fmt.Errorf("Clipboard text is empty")
		}
		return nil
	})
	return
}
func (System) Write(text string) error {
	data, e := syscall.UTF16FromString(text)
	if e != nil {
		return e
	}
	return session(func() error {
		h, _, e := alloc.Call(0x42, uintptr(len(data)*2))
		if h == 0 {
			return fmt.Errorf("allocate Clipboard: %v", e)
		}
		owned := true
		defer func() {
			if owned {
				free.Call(h)
			}
		}()
		p, _, e := lock.Call(h)
		if p == 0 {
			return fmt.Errorf("lock Clipboard allocation: %v", e)
		}
		moveMemory.Call(p, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
		runtime.KeepAlive(data)
		unlock.Call(h)
		ok, _, e := empty.Call()
		if ok == 0 {
			return fmt.Errorf("empty Clipboard: %v", e)
		}
		ok, _, e = set.Call(13, h)
		if ok == 0 {
			return fmt.Errorf("write Clipboard: %v", e)
		}
		owned = false
		return nil
	})
}
