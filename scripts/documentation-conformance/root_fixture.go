// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

// Root installation is its exact published shell procedure in a native PTY.
// The headless request uses the same production CLI fixture as the reference.
const rootCommandTest = `
func TestRootCommands(t *testing.T){
 parent:=t.TempDir();clone:=filepath.Join(parent,"hatmax")
 run(t,__SOURCE_ROOT__,"git","clone","--quiet","--no-hardlinks","--no-checkout",__SOURCE_ROOT__,clone)
 run(t,clone,"git","checkout","--quiet","--detach",__ROOT_HEAD__)
 start:=filepath.Join(parent,"myapp");must(t,os.Mkdir(start,0700))
 t.Setenv("XDG_STATE_HOME",t.TempDir());t.Setenv("XDG_CACHE_HOME",t.TempDir())
 directory,err:=os.Getwd();must(t,err);t.Setenv("GOBIN",directory)
 terminalBound(t,start,__ROOT_INSTALL__,2*time.Minute)
 if !strings.Contains(run(t,clone,"git","rev-parse","HEAD"),__ROOT_HEAD__){t.Fatal("root clone head mismatch")}
 t.Log("root: exact source-install and TUI procedure; native PTY help/quit; no model inference")
}
`
