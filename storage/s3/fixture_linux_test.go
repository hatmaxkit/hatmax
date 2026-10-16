//go:build integration && linux

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"os/exec"
	"runtime"
	"syscall"
)

// Keep the creating thread alive through Wait. Linux's parent-death signal then
// also removes the owned server if a test timeout or panic exits the harness.
func startFixtureProcess(command *exec.Cmd) (chan error, error) {
	wait := make(chan error, 1)
	started := make(chan error, 1)
	command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}

	go func() {
		runtime.LockOSThread()

		defer runtime.UnlockOSThread()

		err := command.Start()
		started <- err

		if err == nil {
			wait <- command.Wait()
		}
	}()

	return wait, <-started
}
