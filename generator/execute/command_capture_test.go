// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Both execution paths must drain noisy stdout and stderr, retain the leading
// context and final failure summary, and preserve failure classification.
func TestCommandCapture(t *testing.T) {
	root := installCaptureChild(t)

	for _, runner := range []string{"repository", "staging"} {
		for _, mode := range []string{"quiet", "success", "failure", "infrastructure"} {
			t.Run(runner+"/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				command := captureChildCommand(mode)

				var (
					output string
					err    error
				)

				if runner == "repository" {
					var evidence CommandEvidence

					evidence, err = runRepositoryCommand(ctx, root, command)
					output = evidence.Output

					wantExit := 0
					if mode == "failure" || mode == "infrastructure" {
						wantExit = 7
					}

					if evidence.ExitCode != wantExit {
						t.Fatalf("exit code: %d; want %d (%v)", evidence.ExitCode, wantExit, err)
					}
				} else {
					results, runErr := validateApplicationStaging(ctx, root, []Command{command, captureChildCommand("quiet")})
					err = runErr
					wantStatus := ValidationCommandPassed

					wantCount := 2
					if mode == "failure" {
						wantStatus, wantCount = ValidationCommandFailed, 1
					} else if mode == "infrastructure" {
						wantStatus = ValidationCommandIncomplete
					}

					if len(results) != wantCount || results[0].Status != wantStatus {
						t.Fatalf("staging results: %+v, %v", results, err)
					}

					output = results[0].Output
				}

				wantError := mode == "failure" || (mode == "infrastructure" && runner == "repository")
				if (err != nil) != wantError {
					t.Fatalf("error: %v; want failure=%v", err, wantError)
				}

				if runner == "staging" && (mode == "quiet" || mode == "success") {
					if output != "" {
						t.Fatalf("successful staging retained output: %q", output)
					}

					return
				}

				if mode == "quiet" {
					if output != "short output\n" {
						t.Fatalf("quiet output: %q", output)
					}

					return
				}

				if len(output) > maximumCommandEvidenceBytes || !strings.Contains(output, "[output truncated]") || !strings.HasPrefix(output, "capture beginning\n") || !strings.HasSuffix(strings.TrimSpace(output), "final command summary") {
					t.Fatalf("capture: %d bytes, head=%v, tail=%v, truncated=%v", len(output), strings.HasPrefix(output, "capture beginning\n"), strings.HasSuffix(output, "final command summary\n"), strings.Contains(output, "[output truncated]"))
				}

				if mode == "infrastructure" && strings.Contains(strings.ToLower(output), "testcontainers") {
					t.Fatal("infrastructure fixture marker was not in the discarded middle")
				}
			})
		}
	}
}

// Cancellation must stop each runner after output exceeds the capture budget,
// without waiting for the bounded child sleep or losing failure evidence.
func TestCaptureCanceled(t *testing.T) {
	root := installCaptureChild(t)
	for _, runner := range []string{"repository", "staging"} {
		t.Run(runner, func(t *testing.T) {
			ready := filepath.Join(t.TempDir(), "ready")
			t.Setenv("HATMAX_CAPTURE_READY", ready)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			done := make(chan error, 1)

			go func() {
				if runner == "repository" {
					evidence, err := runRepositoryCommand(ctx, root, captureChildCommand("cancel"))
					if evidence.ExitCode == 0 || len(evidence.Output) > maximumCommandEvidenceBytes || !strings.Contains(evidence.Output, "[output truncated]") {
						err = fmt.Errorf("canceled evidence: code=%d bytes=%d", evidence.ExitCode, len(evidence.Output))
					}

					done <- err

					return
				}

				results, err := validateApplicationStaging(ctx, root, []Command{captureChildCommand("cancel")})
				if len(results) != 1 || results[0].Status != ValidationCommandFailed || len(results[0].Output) > maximumCommandEvidenceBytes {
					err = fmt.Errorf("canceled staging evidence: %+v", results)
				}

				done <- err
			}()

			for {
				_, err := os.Stat(ready)
				if err == nil {
					break
				}

				select {
				case err := <-done:
					t.Fatalf("child stopped before cancellation: %v", err)
				case <-ctx.Done():
					<-done
					t.Fatal("child did not reach cancellation point")
				case <-time.After(5 * time.Millisecond):
				}
			}

			cancel()

			select {
			case err := <-done:
				if err == nil {
					t.Fatal("canceled command reported success")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("canceled runner did not return")
			}
		})
	}
}

func installCaptureChild(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink(executable, filepath.Join(root, "capture-test"))
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HATMAX_CAPTURE_CHILD", "1")
	t.Setenv("GORACE", "atexit_sleep_ms=0")
	t.Setenv("GOCOVERDIR", t.TempDir())

	return root
}

func captureChildCommand(mode string) Command {
	return Command{Name: "validation.test", Args: []string{"capture-test", "-test.run=^TestCaptureChild$", "--", mode}}
}

// Subprocess fixture only: emit a finite 2 MiB stream split between stdout and
// stderr, then exit or wait long enough for deterministic parent cancellation.
func TestCaptureChild(t *testing.T) {
	if os.Getenv("HATMAX_CAPTURE_CHILD") != "1" {
		return
	}

	mode := os.Args[len(os.Args)-1]
	if mode == "quiet" {
		fmt.Fprintln(os.Stdout, "short output")
		os.Exit(0)
	}

	fmt.Fprintln(os.Stdout, "capture beginning")

	chunk := strings.Repeat("x", 1024)

	for i := 0; i < 2048; i++ {
		var writer io.Writer = os.Stdout
		if i%2 != 0 {
			writer = os.Stderr
		}

		fmt.Fprint(writer, chunk)

		if i == 1024 && mode == "infrastructure" {
			fmt.Fprintln(writer, "TeStCoNtAiNeRs: unavailable")
		}
	}

	fmt.Fprintln(os.Stderr, "final command summary")

	if mode == "cancel" {
		err := os.WriteFile(os.Getenv("HATMAX_CAPTURE_READY"), nil, 0o600)
		if err != nil {
			os.Exit(9)
		}

		time.Sleep(30 * time.Second)
	}

	if mode == "failure" || mode == "infrastructure" {
		os.Exit(7)
	}

	os.Exit(0)
}
