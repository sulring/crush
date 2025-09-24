//go:build windows
// +build windows

package cmd

import (
	"os/exec"
)

func detachProcess(c *exec.Cmd, stdoutPath, stderrPath string) {
	argv1 := c.Args[0]
	c.Path = "cmd"
	c.Args = []string{
		"cmd",
		"/c",
		argv1,
		">",
		stdoutPath,
		"2>",
		stderrPath,
	}
}
