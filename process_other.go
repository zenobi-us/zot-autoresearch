//go:build !unix && !windows

package main

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
