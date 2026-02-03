//go:build !windows
// +build !windows

package executor

import (
	"os/exec"
	"syscall"
)

func setCmdProcessAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
