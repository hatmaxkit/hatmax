// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"hatmax.adrianpk.com/storage"
)

// Concurrent signing must not mutate SDK state shared with another signature
// or object deletion. Every returned grant retains its own exact policy.
func TestConcurrentSigning(t *testing.T) {
	cases := []struct {
		name     string
		signers  int
		deleters int
	}{
		{"signatures", 4, 0},
		{"deletions", 2, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodDelete {
					w.WriteHeader(http.StatusMethodNotAllowed)

					return
				}

				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			options := testOptions(t, server.URL)
			options.PostPolicy = PostPolicyMinIO
			store := newTestStore(t, options)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			const iterations = 32

			requests := make([]storage.UploadRequest, tc.signers*iterations)
			grants := make([]storage.UploadGrant, len(requests))
			failures := make(chan error, tc.signers+tc.deleters)
			start := make(chan struct{})

			var workers sync.WaitGroup

			for worker := range tc.signers + tc.deleters {
				workers.Add(1)

				go func() {
					defer workers.Done()

					<-start

					for step := range iterations {
						key := fmt.Sprintf("staging/worker-%d-object-%d", worker, step)
						if worker >= tc.signers {
							err := store.Delete(ctx, key)
							if err != nil {
								failures <- err

								return
							}

							continue
						}

						index := worker*iterations + step
						request := storage.UploadRequest{Key: key, ContentType: "text/plain", MinBytes: 1, MaxBytes: int64(step + 1), ExpiresAt: time.Now().Add(time.Minute)}

						grant, err := store.SignUpload(ctx, request)
						if err != nil {
							failures <- err

							return
						}

						requests[index], grants[index] = request, grant
					}
				}()
			}

			close(start)
			workers.Wait()
			close(failures)

			for err := range failures {
				t.Error(err)
			}

			if t.Failed() {
				return
			}

			for index, grant := range grants {
				assertPolicy(t, grant, requests[index], options.Bucket)
			}

			assertEmptySpool(t, options.SpoolDirectory)
		})
	}
}
