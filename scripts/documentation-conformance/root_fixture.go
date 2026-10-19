// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

// Source installation follows the User Guide's shell procedure in a native PTY.
// The headless request uses the same production CLI fixture as the reference.
const sourceInstallTest = `
func TestSourceInstall(t *testing.T){
 parent:=t.TempDir();clone:=filepath.Join(parent,"hatmax")
 run(t,__SOURCE_ROOT__,"git","clone","--quiet","--no-hardlinks","--no-checkout",__SOURCE_ROOT__,clone)
 run(t,clone,"git","checkout","--quiet","--detach",__SOURCE_HEAD__)
 t.Setenv("XDG_STATE_HOME",t.TempDir());t.Setenv("XDG_CACHE_HOME",t.TempDir())
 directory,err:=os.Getwd();must(t,err);t.Setenv("GOBIN",directory)
 terminalBound(t,clone,__SOURCE_INSTALL__,2*time.Minute)
 if !strings.Contains(run(t,clone,"git","rev-parse","HEAD"),__SOURCE_HEAD__){t.Fatal("source clone head mismatch")}
 t.Log("User Guide: exact source-install and TUI procedure; native PTY help/quit; no model inference")
}
`
