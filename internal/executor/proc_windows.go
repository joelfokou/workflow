//go:build windows
// +build windows

package executor

import "os/exec"

func setCmdProcessAttrs(cmd *exec.Cmd) {
	// No special process attributes needed for Windows
}
