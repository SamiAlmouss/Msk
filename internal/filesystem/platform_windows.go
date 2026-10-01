//go:build windows

package filesystem

import (
	"os"
	"strings"
	"syscall"
)

func isLink(path string, info os.FileInfo) (bool, error) {
	p, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return false, e
	}
	attr, e := syscall.GetFileAttributes(p)
	if e != nil {
		return false, e
	}
	return info.Mode()&os.ModeSymlink != 0 || attr&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
func samePath(a, b string) bool { return strings.EqualFold(a, b) }
