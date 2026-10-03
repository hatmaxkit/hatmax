// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const (
	passwordSaltBytes          = 16
	passwordKeyBytes           = 32
	passwordRecordBytes        = 128
	passwordInputBytes         = 4096
	passwordInputRunes         = 1024
	passwordMinMemoryKiB       = 19 * 1024
	passwordMaxMemoryKiB       = 256 * 1024
	passwordMaxIterations      = 10
	passwordMaxParallelism     = 8
	passwordMaxConcurrent      = 16
	passwordMaxActiveMemoryKiB = 512 * 1024
)

var (
	ErrPasswordVerifierConfig = errors.New("invalid password verifier configuration")
	ErrPasswordInput          = errors.New("invalid password input")
	ErrPasswordRecord         = errors.New("invalid password record")
	ErrPasswordMismatch       = errors.New("password mismatch")
	ErrPasswordVerifierBusy   = errors.New("password verifier busy")
)

// PasswordVerifierConfig selects Argon2id creation parameters and the upper
// bounds accepted from stored records. Memory is in KiB. Zero values default to
// 65536 KiB, three iterations, four lanes, the same verification maxima and two
// concurrent KDF operations. MaxMemoryKiB * MaxConcurrent cannot exceed 512 MiB.
type PasswordVerifierConfig struct {
	MemoryKiB      uint32
	Iterations     uint32
	Parallelism    uint8
	MaxMemoryKiB   uint32
	MaxIterations  uint32
	MaxParallelism uint8
	MaxConcurrent  int
}

// PasswordVerifier creates and verifies PHC Argon2id v=19 records. Share one
// instance across an application's credential operations: its immutable
// configuration and admission slots bound active work, without a waiting queue.
// It performs no database operations and does not enforce new-password policy.
type PasswordVerifier struct {
	config PasswordVerifierConfig
	slots  chan struct{}
}

// NewPasswordVerifier validates creation and verification resource limits.
// Supported memory is 19 through 256 MiB, iterations 1 through 10 and lanes 1
// through 8. Below 64 MiB, at least two iterations are required. Admission permits
// 1 through 16 operations, within the aggregate memory limit.
func NewPasswordVerifier(cfg PasswordVerifierConfig) (*PasswordVerifier, error) {
	if cfg.MemoryKiB == 0 {
		cfg.MemoryKiB = 64 * 1024
	}

	if cfg.Iterations == 0 {
		cfg.Iterations = 3
	}

	if cfg.Parallelism == 0 {
		cfg.Parallelism = 4
	}

	if cfg.MaxMemoryKiB == 0 {
		cfg.MaxMemoryKiB = cfg.MemoryKiB
	}

	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = cfg.Iterations
	}

	if cfg.MaxParallelism == 0 {
		cfg.MaxParallelism = cfg.Parallelism
	}

	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 2
	}

	if !validPasswordCost(cfg.MemoryKiB, cfg.Iterations, cfg.Parallelism) {
		return nil, fmt.Errorf("%w: creation cost", ErrPasswordVerifierConfig)
	}

	if !validPasswordCost(cfg.MaxMemoryKiB, cfg.MaxIterations, cfg.MaxParallelism) {
		return nil, fmt.Errorf("%w: verification limits", ErrPasswordVerifierConfig)
	}

	if cfg.MaxMemoryKiB < cfg.MemoryKiB || cfg.MaxIterations < cfg.Iterations || cfg.MaxParallelism < cfg.Parallelism {
		return nil, fmt.Errorf("%w: creation exceeds verification limits", ErrPasswordVerifierConfig)
	}

	if cfg.MaxConcurrent < 1 || cfg.MaxConcurrent > passwordMaxConcurrent {
		return nil, fmt.Errorf("%w: concurrency", ErrPasswordVerifierConfig)
	}

	if uint64(cfg.MaxMemoryKiB)*uint64(cfg.MaxConcurrent) > passwordMaxActiveMemoryKiB {
		return nil, fmt.Errorf("%w: aggregate memory", ErrPasswordVerifierConfig)
	}

	return &PasswordVerifier{config: cfg, slots: make(chan struct{}, cfg.MaxConcurrent)}, nil
}

// Hash NFC-normalizes valid UTF-8 and creates a record with an independent
// random 16-byte salt and 32-byte output. Input is limited to 4096 bytes before
// and after normalization, and 1024 normalized code points. Whitespace and case
// are preserved. Apply auth.PasswordPolicy before creating a new credential.
// Failure returns an empty record. Context is checked before and after the KDF;
// Argon2 cannot be interrupted, so the slot stays occupied until it returns.
func (v *PasswordVerifier) Hash(ctx context.Context, password string) (string, error) {
	password, err := prepareVerifierPassword(password)
	if err != nil {
		return "", err
	}

	err = v.acquire(ctx)
	if err != nil {
		return "", err
	}
	defer v.release()

	salt := make([]byte, passwordSaltBytes)

	_, err = rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("password salt: %w", err)
	}

	key, err := derivePasswordKey(ctx, password, salt, v.config.MemoryKiB, v.config.Iterations, v.config.Parallelism)
	if err != nil {
		return "", err
	}
	defer clear(key)

	return encodePasswordRecord(v.config.MemoryKiB, v.config.Iterations, v.config.Parallelism, salt, key), nil
}

// Verify returns nil on a match, ErrPasswordMismatch on a valid nonmatching
// record, and ErrPasswordRecord for malformed, unsupported or excessive records.
// Other errors indicate invalid input/configuration, busy admission or context
// failure. It uses the same NFC input processing as Hash, without rerunning
// new-password policy. It never reads bcrypt or upgrades stored credentials.
func (v *PasswordVerifier) Verify(ctx context.Context, record, password string) error {
	if v == nil || v.slots == nil {
		return ErrPasswordVerifierConfig
	}

	parsed, err := parsePasswordRecord(record, v.config)
	if err != nil {
		return err
	}

	password, err = prepareVerifierPassword(password)
	if err != nil {
		return err
	}

	err = v.acquire(ctx)
	if err != nil {
		return err
	}
	defer v.release()

	key, err := derivePasswordKey(ctx, password, parsed.salt, parsed.memory, parsed.iterations, parsed.parallelism)
	if err != nil {
		return err
	}
	defer clear(key)

	if subtle.ConstantTimeCompare(key, parsed.key) != 1 {
		return ErrPasswordMismatch
	}

	return nil
}

func (v *PasswordVerifier) acquire(ctx context.Context) error {
	if v == nil || v.slots == nil {
		return ErrPasswordVerifierConfig
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	select {
	case v.slots <- struct{}{}:
		return nil
	default:
		return ErrPasswordVerifierBusy
	}
}

func (v *PasswordVerifier) release() {
	<-v.slots
}

func prepareVerifierPassword(password string) (string, error) {
	if len(password) > passwordInputBytes || !utf8.ValidString(password) {
		return "", ErrPasswordInput
	}

	password = norm.NFC.String(password)
	if len(password) > passwordInputBytes || utf8.RuneCountInString(password) > passwordInputRunes {
		return "", ErrPasswordInput
	}

	return password, nil
}

func derivePasswordKey(ctx context.Context, password string, salt []byte, memory, iterations uint32, parallelism uint8) ([]byte, error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	input := []byte(password)
	defer clear(input)

	key := argon2.IDKey(input, salt, iterations, memory, parallelism, passwordKeyBytes)

	err = ctx.Err()
	if err != nil {
		clear(key)

		return nil, err
	}

	return key, nil
}

func validPasswordCost(memory, iterations uint32, parallelism uint8) bool {
	return memory >= passwordMinMemoryKiB && memory <= passwordMaxMemoryKiB &&
		iterations >= 1 && iterations <= passwordMaxIterations &&
		parallelism >= 1 && parallelism <= passwordMaxParallelism &&
		(memory >= 64*1024 || iterations >= 2)
}

type passwordRecord struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	salt        []byte
	key         []byte
}

func parsePasswordRecord(record string, cfg PasswordVerifierConfig) (passwordRecord, error) {
	if len(record) > passwordRecordBytes {
		return passwordRecord{}, ErrPasswordRecord
	}

	fields := strings.Split(record, "$")
	if len(fields) != 6 || fields[0] != "" || fields[1] != "argon2id" || fields[2] != "v=19" {
		return passwordRecord{}, ErrPasswordRecord
	}

	parameters := strings.Split(fields[3], ",")
	if len(parameters) != 3 {
		return passwordRecord{}, ErrPasswordRecord
	}

	memory, err := parsePasswordParameter(parameters[0], "m=", 32)
	if err != nil {
		return passwordRecord{}, err
	}

	iterations, err := parsePasswordParameter(parameters[1], "t=", 32)
	if err != nil {
		return passwordRecord{}, err
	}

	parallelism, err := parsePasswordParameter(parameters[2], "p=", 8)
	if err != nil {
		return passwordRecord{}, err
	}

	if !validPasswordCost(uint32(memory), uint32(iterations), uint8(parallelism)) ||
		memory > uint64(cfg.MaxMemoryKiB) || iterations > uint64(cfg.MaxIterations) || parallelism > uint64(cfg.MaxParallelism) {
		return passwordRecord{}, ErrPasswordRecord
	}

	salt, err := decodePasswordField(fields[4], passwordSaltBytes)
	if err != nil {
		return passwordRecord{}, err
	}

	key, err := decodePasswordField(fields[5], passwordKeyBytes)
	if err != nil {
		return passwordRecord{}, err
	}

	return passwordRecord{memory: uint32(memory), iterations: uint32(iterations), parallelism: uint8(parallelism), salt: salt, key: key}, nil
}

func parsePasswordParameter(field, prefix string, bits int) (uint64, error) {
	if !strings.HasPrefix(field, prefix) {
		return 0, ErrPasswordRecord
	}

	digits := strings.TrimPrefix(field, prefix)
	if digits == "" || (len(digits) > 1 && digits[0] == '0') {
		return 0, ErrPasswordRecord
	}

	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return 0, ErrPasswordRecord
		}
	}

	value, err := strconv.ParseUint(digits, 10, bits)
	if err != nil {
		return 0, ErrPasswordRecord
	}

	return value, nil
}

func decodePasswordField(field string, size int) ([]byte, error) {
	if len(field) != base64.RawStdEncoding.EncodedLen(size) {
		return nil, ErrPasswordRecord
	}

	value, err := base64.RawStdEncoding.Strict().DecodeString(field)
	if err != nil || len(value) != size || base64.RawStdEncoding.EncodeToString(value) != field {
		return nil, ErrPasswordRecord
	}

	return value, nil
}

func encodePasswordRecord(memory, iterations uint32, parallelism uint8, salt, key []byte) string {
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, iterations, parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}
