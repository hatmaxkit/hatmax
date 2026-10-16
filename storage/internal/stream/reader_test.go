// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package stream

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"hatmax.adrianpk.com/storage"
)

type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) { return 0, nil }

// TestReads prevents truncation success, busy loops and reads after terminal errors.
func TestReads(t *testing.T) {
	cases := []struct {
		name  string
		input io.Reader
		want  string
		err   error
	}{
		{"empty", strings.NewReader(""), "", nil},
		{"exact", strings.NewReader("1234"), "1234", nil},
		{"overflow", strings.NewReader("12345"), "1234", storage.ErrTooLarge},
		{"no progress", emptyReader{}, "", io.ErrNoProgress},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewReader(context.Background(), tc.input, 4)

			n, err := reader.Read(nil)
			if n != 0 || err != nil {
				t.Fatalf("zero-length read=%d/%v", n, err)
			}

			data, err := io.ReadAll(reader)
			if string(data) != tc.want || !errors.Is(err, tc.err) {
				t.Fatalf("read=%q/%v", data, err)
			}

			_, err = reader.Read(make([]byte, 1))
			if tc.err != nil && !errors.Is(err, tc.err) || tc.err == nil && !errors.Is(err, io.EOF) {
				t.Fatalf("terminal read=%v", err)
			}
		})
	}
}
