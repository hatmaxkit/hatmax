// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"
	"strings"
	"testing"
)

// Boundary and fragmented writes must retain exact short output or the same
// bounded head/tail, regardless of how os/exec divides pipe reads.
func TestOutputBudget(t *testing.T) {
	for _, size := range []int{0, 1, maximumCommandEvidenceBytes - 1, maximumCommandEvidenceBytes, maximumCommandEvidenceBytes + 1, 3 * maximumCommandEvidenceBytes} {
		for _, chunk := range []int{1, 17, 4096, 32768} {
			t.Run(fmt.Sprintf("bytes %d/chunk %d", size, chunk), func(t *testing.T) {
				value := strings.Repeat("0123456789", (size+9)/10)[:size]

				var output commandOutput

				for offset := 0; offset < len(value); offset += chunk {
					part := value[offset:min(offset+chunk, len(value))]

					n, err := output.Write([]byte(part))
					if err != nil || n != len(part) {
						t.Fatalf("write: %d, %v; want %d", n, err, len(part))
					}

					if cap(output.head) > maximumCommandEvidenceBytes || cap(output.tail) > commandOutputTailBytes {
						t.Fatalf("unbounded storage: head=%d tail=%d", cap(output.head), cap(output.tail))
					}
				}

				want := value
				if size > maximumCommandEvidenceBytes {
					want = value[:commandOutputHeadBytes] + commandOutputTruncation + value[len(value)-commandOutputTailBytes:]
				}

				if output.String() != want || len(output.String()) > maximumCommandEvidenceBytes || output.truncated != (size > maximumCommandEvidenceBytes) {
					t.Fatalf("capture: bytes=%d, truncated=%v; want bytes=%d", len(output.String()), output.truncated, len(want))
				}

				n, err := output.Write(nil)
				if n != 0 || err != nil || output.String() != want {
					t.Fatalf("empty write changed capture: %d, %v", n, err)
				}
			})
		}
	}
}

// Infrastructure classification must inspect discarded bytes and recognize
// existing markers even across writes or bounded scan-window boundaries.
func TestOutputInfrastructure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		parts []string
		check bool
		want  bool
	}{
		{name: "split marker", parts: []string{"cannot connect to the ", "docker daemon"}, check: true, want: true},
		{name: "mixed case", parts: []string{"TeStCoNtAiNeRs"}, check: true, want: true},
		{name: "scan boundary", parts: []string{strings.Repeat("x", maximumCommandEvidenceBytes-5) + "testcontainers"}, check: true, want: true},
		{name: "discarded middle", parts: []string{strings.Repeat("x", 2*maximumCommandEvidenceBytes), "connection refused", strings.Repeat("x", 2*maximumCommandEvidenceBytes)}, check: true, want: true},
		{name: "no marker", parts: []string{"assertion failed"}, check: true},
		{name: "not requested", parts: []string{"testcontainers"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := commandOutput{checkInfrastructure: tc.check}
			for _, part := range tc.parts {
				_, err := output.Write([]byte(part))
				if err != nil {
					t.Fatal(err)
				}
			}

			if output.infrastructureUnavailable != tc.want || output.scanSize > infrastructureScanBytes {
				t.Fatalf("classification=%v scan=%d; want %v", output.infrastructureUnavailable, output.scanSize, tc.want)
			}
		})
	}
}
