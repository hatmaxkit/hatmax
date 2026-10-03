// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// Trusted configuration cannot exceed either per-operation or aggregate budgets,
// or create records that the same verifier would refuse to read.
func TestPasswordVerifierConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cfg   PasswordVerifierConfig
		valid bool
	}{
		{name: "defaults", valid: true},
		{name: "minimum cost", cfg: PasswordVerifierConfig{MemoryKiB: 19456, Iterations: 2, Parallelism: 1}, valid: true},
		{name: "maximum envelope", cfg: PasswordVerifierConfig{MemoryKiB: 262144, Iterations: 10, Parallelism: 8, MaxConcurrent: 2}, valid: true},
		{name: "below memory floor", cfg: PasswordVerifierConfig{MemoryKiB: 19455}},
		{name: "weak small memory", cfg: PasswordVerifierConfig{MemoryKiB: 19456, Iterations: 1}},
		{name: "excessive memory", cfg: PasswordVerifierConfig{MemoryKiB: 262145}},
		{name: "integer memory limit", cfg: PasswordVerifierConfig{MemoryKiB: 1 << 31}},
		{name: "excessive iterations", cfg: PasswordVerifierConfig{Iterations: 11}},
		{name: "excessive lanes", cfg: PasswordVerifierConfig{Parallelism: 9}},
		{name: "negative concurrency", cfg: PasswordVerifierConfig{MaxConcurrent: -1}},
		{name: "excessive concurrency", cfg: PasswordVerifierConfig{MaxConcurrent: 17}},
		{name: "aggregate memory", cfg: PasswordVerifierConfig{MaxConcurrent: 9}},
		{name: "smaller memory ceiling", cfg: PasswordVerifierConfig{MaxMemoryKiB: 19456}},
		{name: "smaller time ceiling", cfg: PasswordVerifierConfig{MaxIterations: 2}},
		{name: "smaller lane ceiling", cfg: PasswordVerifierConfig{MaxParallelism: 1}},
		{name: "excessive memory ceiling", cfg: PasswordVerifierConfig{MaxMemoryKiB: 262145}},
		{name: "excessive time ceiling", cfg: PasswordVerifierConfig{MaxIterations: 11}},
		{name: "excessive lane ceiling", cfg: PasswordVerifierConfig{MaxParallelism: 9}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewPasswordVerifier(tc.cfg)
			if tc.valid {
				if err != nil || v == nil {
					t.Fatalf("valid configuration rejected: %v", err)
				}

				return
			}

			if !errors.Is(err, ErrPasswordVerifierConfig) || v != nil {
				t.Fatalf("invalid configuration = %v, %v", v, err)
			}
		})
	}
}

// Every parser rejection occurs before admission/KDF, including algorithms that
// would otherwise select a legacy reader and parameters that could exhaust memory.
func TestPasswordRecordBounds(t *testing.T) {
	v := newTestPasswordVerifier(t)
	salt := base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef"))
	key := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	good := encodePasswordRecord(19456, 2, 1, []byte("0123456789abcdef"), make([]byte, 32))

	for range cap(v.slots) {
		v.slots <- struct{}{}
	}

	for _, tc := range []struct{ name, record string }{
		{"empty", ""},
		{"oversized", strings.Repeat("x", 129)},
		{"bcrypt", "$2a$12$legacy"},
		{"argon2i", strings.Replace(good, "argon2id", "argon2i", 1)},
		{"unknown algorithm", strings.Replace(good, "argon2id", "unknown", 1)},
		{"old version", strings.Replace(good, "v=19", "v=16", 1)},
		{"unknown version", strings.Replace(good, "v=19", "v=20", 1)},
		{"missing version", strings.Replace(good, "v=19$", "", 1)},
		{"extra field", good + "$extra"},
		{"missing leading separator", strings.TrimPrefix(good, "$")},
		{"zero memory", strings.Replace(good, "m=19456", "m=0", 1)},
		{"small memory", strings.Replace(good, "m=19456", "m=19455", 1)},
		{"absolute memory cap", strings.Replace(good, "m=19456", "m=262145", 1)},
		{"configured memory cap", strings.Replace(good, "m=19456", "m=65536", 1)},
		{"memory overflow", strings.Replace(good, "m=19456", "m=4294967296", 1)},
		{"negative memory", strings.Replace(good, "m=19456", "m=-19456", 1)},
		{"signed memory", strings.Replace(good, "m=19456", "m=+19456", 1)},
		{"memory padding", strings.Replace(good, "m=19456", "m=019456", 1)},
		{"memory spaces", strings.Replace(good, "m=19456", "m=19456 ", 1)},
		{"empty memory", strings.Replace(good, "m=19456", "m=", 1)},
		{"zero iterations", strings.Replace(good, "t=2", "t=0", 1)},
		{"weak iterations", strings.Replace(good, "t=2", "t=1", 1)},
		{"absolute time cap", strings.Replace(good, "t=2", "t=11", 1)},
		{"configured time cap", strings.Replace(good, "t=2", "t=3", 1)},
		{"time overflow", strings.Replace(good, "t=2", "t=4294967296", 1)},
		{"zero lanes", strings.Replace(good, "p=1", "p=0", 1)},
		{"absolute lane cap", strings.Replace(good, "p=1", "p=9", 1)},
		{"configured lane cap", strings.Replace(good, "p=1", "p=2", 1)},
		{"lane overflow", strings.Replace(good, "p=1", "p=256", 1)},
		{"parameter order", strings.Replace(good, "m=19456,t=2", "t=2,m=19456", 1)},
		{"duplicate parameter", strings.Replace(good, "t=2", "m=19456", 1)},
		{"extra parameter", strings.Replace(good, "p=1", "p=1,x=1", 1)},
		{"short salt", strings.Replace(good, salt, salt[:21], 1)},
		{"long salt", strings.Replace(good, salt, salt+"A", 1)},
		{"padded salt", strings.Replace(good, salt, salt+"==", 1)},
		{"invalid salt", strings.Replace(good, salt, "!"+salt[1:], 1)},
		{"noncanonical salt", strings.Replace(good, salt, salt[:21]+"h", 1)},
		{"short key", strings.Replace(good, key, key[:42], 1)},
		{"long key", strings.Replace(good, key, key+"A", 1)},
		{"padded key", strings.Replace(good, key, key+"=", 1)},
		{"noncanonical key", strings.Replace(good, key, key[:42]+"B", 1)},
		{"invalid key", strings.Replace(good, key, "!"+key[1:], 1)},
		{"embedded newline", strings.Replace(good, salt, salt[:5]+"\n"+salt[6:], 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Verify(context.Background(), tc.record, "candidate")
			if !errors.Is(err, ErrPasswordRecord) {
				t.Fatalf("verification = %v; want invalid record", err)
			}
		})
	}
}

// The resource ceilings are inclusive. Parsing at each ceiling proves accepted
// bounds without allocating a maximum-cost Argon2 buffer in a unit test.
func TestPasswordRecordCeilings(t *testing.T) {
	cfg := PasswordVerifierConfig{MaxMemoryKiB: 262144, MaxIterations: 10, MaxParallelism: 8}

	for _, tc := range []struct {
		name               string
		memory, iterations uint32
		lanes              uint8
	}{
		{"minimum", 19456, 2, 1},
		{"defaults", 65536, 3, 4},
		{"maximum", 262144, 10, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := encodePasswordRecord(tc.memory, tc.iterations, tc.lanes, make([]byte, 16), make([]byte, 32))

			parsed, err := parsePasswordRecord(record, cfg)
			if err != nil || parsed.memory != tc.memory || parsed.iterations != tc.iterations || parsed.parallelism != tc.lanes {
				t.Fatalf("ceiling rejected: %v", err)
			}
		})
	}
}

// Input is bounded before normalization and after expansion. Verification does
// not reapply candidate minimum/blocklist policy; it only processes format input.
func TestPasswordVerifierInput(t *testing.T) {
	v := newTestPasswordVerifier(t)
	for range cap(v.slots) {
		v.slots <- struct{}{}
	}

	good := encodePasswordRecord(19456, 2, 1, make([]byte, 16), make([]byte, 32))

	for _, tc := range []struct {
		name, password string
		invalid        bool
	}{
		{"invalid utf8", "\xff", true},
		{"raw bytes", strings.Repeat("a", 4097), true},
		{"code points", strings.Repeat("a", 1025), true},
		{"normalization expansion", strings.Repeat("\u0344", 513), true},
		{"maximum unicode", strings.Repeat("😀", 1024), false},
		{"normalized unicode", strings.Repeat("e\u0301", 1024), false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := ErrPasswordVerifierBusy
			if tc.invalid {
				want = ErrPasswordInput
			}

			record, err := v.Hash(context.Background(), tc.password)
			if record != "" || !errors.Is(err, want) {
				t.Fatalf("hash error = %v; want %v and empty record", err, want)
			}

			err = v.Verify(context.Background(), good, tc.password)
			if !errors.Is(err, want) {
				t.Fatalf("verify error = %v; want %v", err, want)
			}
		})
	}
}

// Canceled requests and full admission fail explicitly. Acquired slots are
// reusable after completion; uninitialized instances cannot bypass limits.
func TestPasswordVerifierAdmission(t *testing.T) {
	v := newTestPasswordVerifier(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deadline, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()

	record := encodePasswordRecord(19456, 2, 1, make([]byte, 16), make([]byte, 32))

	for _, tc := range []struct {
		name string
		ctx  context.Context
		v    *PasswordVerifier
		want error
	}{
		{"canceled", ctx, v, context.Canceled},
		{"deadline", deadline, v, context.DeadlineExceeded},
		{"nil verifier", context.Background(), nil, ErrPasswordVerifierConfig},
		{"zero verifier", context.Background(), &PasswordVerifier{}, ErrPasswordVerifierConfig},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := tc.v.Hash(tc.ctx, "candidate")
			if hash != "" || !errors.Is(err, tc.want) {
				t.Fatalf("hash = %v; want %v", err, tc.want)
			}

			err = tc.v.Verify(tc.ctx, record, "candidate")
			if !errors.Is(err, tc.want) {
				t.Fatalf("verify = %v; want %v", err, tc.want)
			}
		})
	}

	for range cap(v.slots) {
		v.slots <- struct{}{}
	}

	hash, err := v.Hash(context.Background(), "candidate")
	if hash != "" || !errors.Is(err, ErrPasswordVerifierBusy) {
		t.Fatalf("full admission = %v", err)
	}

	err = v.Verify(context.Background(), record, "candidate")
	if !errors.Is(err, ErrPasswordVerifierBusy) {
		t.Fatalf("full verification admission = %v", err)
	}

	for range cap(v.slots) {
		v.release()
	}

	hash, err = v.Hash(context.Background(), "candidate")
	if err != nil {
		t.Fatal(err)
	}

	err = v.Verify(context.Background(), hash, "candidate")
	if err != nil || len(v.slots) != 0 {
		t.Fatalf("slot reuse = %v; active %d", err, len(v.slots))
	}
}

// Parallel callers share admission and salt generation. Saturation may deny work,
// but every admitted operation must finish correctly and release its slot.
func TestPasswordVerifierConcurrent(t *testing.T) {
	v := newTestPasswordVerifier(t)
	results := make(chan string, 16)
	failures := make(chan error, 16)
	start := make(chan struct{})

	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			<-start

			record, err := v.Hash(context.Background(), "concurrent password")
			if err != nil && !errors.Is(err, ErrPasswordVerifierBusy) {
				failures <- err

				return
			}

			if err == nil {
				results <- record
			}
		})
	}

	close(start)
	workers.Wait()
	close(results)
	close(failures)

	for err := range failures {
		t.Fatal(err)
	}

	salts := make(map[string]bool)

	for record := range results {
		parsed, err := parsePasswordRecord(record, v.config)
		if err != nil {
			t.Fatal(err)
		}

		if salts[string(parsed.salt)] {
			t.Fatal("concurrent salt reused")
		}

		salts[string(parsed.salt)] = true

		err = v.Verify(context.Background(), record, "concurrent password")
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(salts) == 0 || len(v.slots) != 0 {
		t.Fatalf("admission failed: completed %d, active %d", len(salts), len(v.slots))
	}
}

type cancelAtCheckContext struct {
	context.Context
	cancel    context.CancelFunc
	remaining int
}

func (c *cancelAtCheckContext) Err() error {
	c.remaining--
	if c.remaining == 0 {
		c.cancel()
	}

	return c.Context.Err()
}

// Cancel deterministically at the completion check, without wall-clock timing.
// Completed KDF work must not approve a late match or leave an admission slot held.
func TestPasswordVerifierLateCancellation(t *testing.T) {
	v := newTestPasswordVerifier(t)

	record, err := v.Hash(context.Background(), "candidate")
	if err != nil {
		t.Fatal(err)
	}

	for _, operation := range []string{"hash", "verify"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			late := &cancelAtCheckContext{Context: ctx, cancel: cancel, remaining: 3}

			var (
				result  string
				lateErr error
			)

			if operation == "hash" {
				result, lateErr = v.Hash(late, "candidate")
			} else {
				lateErr = v.Verify(late, record, "candidate")
			}

			if result != "" || !errors.Is(lateErr, context.Canceled) || len(v.slots) != 0 {
				t.Fatalf("late operation = %v; active slots %d", lateErr, len(v.slots))
			}
		})
	}
}

// Fuzzing exercises only parsing. Even attacker-selected valid maximum parameters
// must not start a KDF, while every accepted encoding must round-trip canonically.
func FuzzPasswordRecord(f *testing.F) {
	for _, record := range []string{
		"", "$2a$12$legacy", "$argon2id$v=19$m=4294967296,t=1,p=255$x$x",
		encodePasswordRecord(19456, 2, 1, make([]byte, 16), make([]byte, 32)),
		encodePasswordRecord(262144, 10, 8, make([]byte, 16), make([]byte, 32)),
	} {
		f.Add(record)
	}

	cfg := PasswordVerifierConfig{MaxMemoryKiB: 262144, MaxIterations: 10, MaxParallelism: 8}

	f.Fuzz(func(t *testing.T, record string) {
		parsed, err := parsePasswordRecord(record, cfg)
		if err != nil {
			if !errors.Is(err, ErrPasswordRecord) {
				t.Fatalf("unclassified parser error: %v", err)
			}

			return
		}

		if !validPasswordCost(parsed.memory, parsed.iterations, parsed.parallelism) || len(parsed.salt) != 16 || len(parsed.key) != 32 {
			t.Fatal("parser accepted invalid bounds")
		}

		if encodePasswordRecord(parsed.memory, parsed.iterations, parsed.parallelism, parsed.salt, parsed.key) != record {
			t.Fatal("parser accepted noncanonical encoding")
		}
	})
}

// Unicode credentials beyond bcrypt's byte limit remain intact, including the
// maximum supported input. Concurrent verification uses the shared admission pool.
func TestPasswordVerifierLongUnicode(t *testing.T) {
	v := newTestPasswordVerifier(t)
	for _, tc := range []struct {
		name   string
		length int
	}{
		{"long unicode", 64},
		{"maximum unicode", 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			password := strings.Repeat("😀", tc.length)

			record, err := v.Hash(context.Background(), password)
			if err != nil {
				t.Fatal(err)
			}

			failures := make(chan error, 2)

			var workers sync.WaitGroup
			for range 2 {
				workers.Go(func() {
					verifyErr := v.Verify(context.Background(), record, password)
					if verifyErr != nil {
						failures <- verifyErr
					}
				})
			}

			workers.Wait()
			close(failures)

			for verifyErr := range failures {
				t.Fatal(verifyErr)
			}
		})
	}
}
