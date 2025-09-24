//go:build !windows
// +build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

func detachProcess(c *exec.Cmd, _, _ string) {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setsid = true
}
