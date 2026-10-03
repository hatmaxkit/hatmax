// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Compare complete verification costs for measured candidate profiles. Record
// creation is outside timing, so salt entropy and setup do not obscure KDF cost.
func BenchmarkPasswordVerifierCost(b *testing.B) {
	for _, tc := range []struct {
		memory, iterations uint32
		lanes              uint8
	}{
		{19456, 2, 1}, {65536, 1, 1}, {65536, 3, 1}, {65536, 3, 4},
	} {
		b.Run(fmt.Sprintf("m%d-t%d-p%d", tc.memory, tc.iterations, tc.lanes), func(b *testing.B) {
			v, err := NewPasswordVerifier(PasswordVerifierConfig{MemoryKiB: tc.memory, Iterations: tc.iterations, Parallelism: tc.lanes})
			if err != nil {
				b.Fatal(err)
			}

			ctx := context.Background()

			record, err := v.Hash(ctx, "benchmark password")
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				err = v.Verify(ctx, record, "benchmark password")
				if err != nil {
					b.Fatal(err)
				}
			}

			b.ReportMetric(float64(tc.memory), "KDF_KiB/op")
		})
	}
}

// Each iteration waits for one finite batch. Worker count and aggregate KDF
// memory are explicit; reported ns/verification is batch-amortized throughput,
// while the standard ns/op measures batch completion, not individual latency.
func BenchmarkPasswordVerifierConcurrency(b *testing.B) {
	for _, count := range []int{1, 2, 4} {
		b.Run(fmt.Sprintf("active%d", count), func(b *testing.B) {
			v, err := NewPasswordVerifier(PasswordVerifierConfig{MaxConcurrent: count})
			if err != nil {
				b.Fatal(err)
			}

			ctx := context.Background()

			record, err := v.Hash(ctx, "benchmark password")
			if err != nil {
				b.Fatal(err)
			}

			failures := make(chan error, count)

			b.ReportAllocs()
			b.ResetTimer()

			started := time.Now()

			for range b.N {
				var workers sync.WaitGroup
				for range count {
					workers.Go(func() {
						verifyErr := v.Verify(ctx, record, "benchmark password")
						if verifyErr != nil {
							failures <- verifyErr
						}
					})
				}

				workers.Wait()

				if len(failures) != 0 {
					b.Fatal(<-failures)
				}
			}

			b.ReportMetric(float64(time.Since(started).Nanoseconds())/float64(b.N*count), "ns/verification")
			b.ReportMetric(float64(v.config.MaxMemoryKiB)*float64(count), "active_KDF_KiB")
		})
	}
}
