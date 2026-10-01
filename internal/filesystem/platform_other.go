//go:build !windows

package filesystem

import "os"

func isLink(_ string, info os.FileInfo) (bool, error) { return info.Mode()&os.ModeSymlink != 0, nil }
func samePath(a, b string) bool                       { return a == b }
