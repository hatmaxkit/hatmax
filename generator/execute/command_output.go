// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

const (
	commandOutputTruncation = "\n[output truncated]\n"
	commandOutputHeadBytes  = (maximumCommandEvidenceBytes - len(commandOutputTruncation)) / 2
	commandOutputTailBytes  = maximumCommandEvidenceBytes - len(commandOutputTruncation) - commandOutputHeadBytes
	// Existing infrastructure markers fit within this cross-write scan window.
	infrastructureScanBytes = 64
)

// commandOutput drains all writes while retaining a bounded head and tail.
// Cmd serializes Write calls when stdout and stderr share this same pointer.
type commandOutput struct {
	head      []byte
	tail      []byte
	truncated bool

	checkInfrastructure       bool
	infrastructureUnavailable bool
	scanTail                  [infrastructureScanBytes]byte
	scanSize                  int
}

func (o *commandOutput) Write(p []byte) (int, error) {
	n := len(p)
	if o.checkInfrastructure && !o.infrastructureUnavailable {
		o.inspectInfrastructure(p)
	}

	if !o.truncated {
		if o.head == nil && n > 0 {
			o.head = make([]byte, 0, maximumCommandEvidenceBytes)
		}

		keep := min(len(p), maximumCommandEvidenceBytes-len(o.head))
		o.head = append(o.head, p[:keep]...)

		p = p[keep:]
		if len(p) == 0 {
			return n, nil
		}

		o.truncated = true
		o.tail = make([]byte, commandOutputTailBytes)
		copy(o.tail, o.head[len(o.head)-commandOutputTailBytes:])
		o.head = o.head[:commandOutputHeadBytes]
	}

	if len(p) >= len(o.tail) {
		copy(o.tail, p[len(p)-len(o.tail):])
	} else {
		copy(o.tail, o.tail[len(p):])
		copy(o.tail[len(o.tail)-len(p):], p)
	}

	// Report the full write as consumed, including discarded bytes, so os/exec
	// keeps draining the pipes instead of returning a short-write error.
	return n, nil
}

func (o *commandOutput) String() string {
	if !o.truncated {
		return string(o.head)
	}

	return string(o.head) + commandOutputTruncation + string(o.tail)
}

func (o *commandOutput) inspectInfrastructure(p []byte) {
	for len(p) > 0 {
		size := min(len(p), maximumCommandEvidenceBytes)

		window := string(o.scanTail[:o.scanSize]) + string(p[:size])
		if testInfrastructureUnavailable(window) {
			o.infrastructureUnavailable = true

			return
		}

		o.scanSize = min(len(window), len(o.scanTail))
		copy(o.scanTail[:], window[len(window)-o.scanSize:])

		p = p[size:]
	}
}
