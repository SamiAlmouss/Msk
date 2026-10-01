//go:build !windows

package clipboard

import "fmt"

func (System) Read() (string, error) {
	return "", fmt.Errorf("Clipboard is supported on Windows; supply a UTF-8 file")
}
func (System) Write(string) error { return fmt.Errorf("Clipboard is supported on Windows; use --save") }
