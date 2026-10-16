// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"hatmax.adrianpk.com/storage"
)

type signedPolicy struct {
	Expiration time.Time         `json:"expiration"`
	Conditions []json.RawMessage `json:"conditions"`
}

func decodePolicy(t *testing.T, grant storage.UploadGrant) signedPolicy {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString(grant.Fields["policy"])
	if err != nil {
		t.Fatal("grant policy is not valid base64")
	}

	var policy signedPolicy

	err = json.Unmarshal(data, &policy)
	if err != nil || !policy.Expiration.Equal(grant.ExpiresAt) {
		t.Fatal("grant metadata differs from actual signed policy expiry")
	}

	return policy
}

func assertPolicy(t *testing.T, grant storage.UploadGrant, request storage.UploadRequest, bucket string) {
	t.Helper()
	policy := decodePolicy(t, grant)

	expected := map[string]string{
		"bucket": bucket, "key": request.Key, "Content-Type": request.ContentType,
		"X-Amz-Algorithm": "AWS4-HMAC-SHA256", "X-Amz-Date": grant.Fields["X-Amz-Date"], "X-Amz-Credential": grant.Fields["X-Amz-Credential"],
	}
	if token := grant.Fields["X-Amz-Security-Token"]; token != "" {
		expected["X-Amz-Security-Token"] = token
	}

	lengthRanges := 0

	for _, raw := range policy.Conditions {
		if len(raw) > 0 && raw[0] == '{' {
			var condition map[string]string

			err := json.Unmarshal(raw, &condition)
			if err != nil || len(condition) != 1 {
				t.Fatal("unexpected policy equality condition")
			}

			for key, value := range condition {
				want, exists := expected[key]
				if !exists || want != value {
					t.Fatalf("unexpected or duplicate policy condition: %s", key)
				}

				delete(expected, key)
			}

			continue
		}

		var (
			condition        []json.RawMessage
			operator         string
			minimum, maximum int64
		)

		err := json.Unmarshal(raw, &condition)
		if err != nil || len(condition) != 3 {
			t.Fatal("unexpected policy operator condition")
		}

		err = json.Unmarshal(condition[0], &operator)
		if err != nil || operator != "content-length-range" {
			t.Fatal("non-exact or unsupported policy operator")
		}

		err = json.Unmarshal(condition[1], &minimum)
		if err != nil {
			t.Fatal("invalid policy minimum")
		}

		err = json.Unmarshal(condition[2], &maximum)
		if err != nil || minimum != request.MinBytes || maximum != request.MaxBytes {
			t.Fatal("policy byte range differs from request")
		}

		lengthRanges++
	}

	if len(expected) != 0 || lengthRanges != 1 || grant.Fields["key"] != request.Key || grant.Fields["Content-Type"] != request.ContentType {
		t.Fatal("required exact target, type or byte policy is missing")
	}

	allowed := map[string]bool{"key": true, "Content-Type": true, "policy": true, "X-Amz-Date": true, "X-Amz-Algorithm": true, "X-Amz-Credential": true, "X-Amz-Signature": true, "X-Amz-Security-Token": true}

	for key := range allowed {
		if key != "X-Amz-Security-Token" && grant.Fields[key] == "" {
			t.Fatalf("missing required form field: %s", key)
		}
	}

	for key, value := range grant.Fields {
		if !allowed[key] || value == "" || strings.Contains(value, "fixture-secret") {
			t.Fatalf("unexpected, empty or unsafe form field: %s", key)
		}
	}
}

// TestSignedPolicy binds the real SDK output to exact conditions without network I/O.
func TestSignedPolicy(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(500)
	}))
	defer server.Close()

	options := testOptions(t, server.URL)
	options.PostPolicy = PostPolicyMinIO
	store := newTestStore(t, options)
	request := storage.UploadRequest{Key: "staging/exact-document", ContentType: "text/plain", MinBytes: 2, MaxBytes: 8, ExpiresAt: time.Now().Add(time.Minute)}

	grant, err := store.SignUpload(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	assertPolicy(t, grant, request, options.Bucket)

	if !grant.ExpiresAt.Equal(request.ExpiresAt.UTC().Truncate(time.Second)) || grant.URL != server.URL+"/"+options.Bucket || requests.Load() != 0 {
		t.Fatal("SDK policy target, absolute expiry or no-I/O contract differs")
	}

	assertEmptySpool(t, options.SpoolDirectory)
}

// TestSigningClock covers precise caller/credential boundaries without wall-clock waits.
func TestSigningClock(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 900000000, time.UTC)
	cases := []struct {
		name       string
		requested  time.Duration
		credential time.Duration
		expiring   bool
		valid      bool
	}{
		{"requested deadline", time.Minute, 0, false, true},
		{"credential deadline", time.Minute, 10 * time.Second, true, true},
		{"later credential", time.Minute, 2 * time.Minute, true, true},
		{"maximum lifetime", 15 * time.Minute, 0, false, true},
		{"excess lifetime", 15*time.Minute + time.Nanosecond, 0, false, false},
		{"expired request", -time.Nanosecond, 0, false, false},
		{"expired credential", time.Minute, -time.Second, true, false},
		{"request rounded away", 50 * time.Millisecond, 0, false, false},
		{"credential rounded away", time.Minute, 50 * time.Millisecond, true, false},
		{"next second survives", 100 * time.Millisecond, 0, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := storage.UploadRequest{Key: "object", ContentType: "text/plain", MaxBytes: 8, ExpiresAt: now.Add(tc.requested)}
			creds := aws.Credentials{AccessKeyID: "fixture-key", SecretAccessKey: "fixture-secret", CanExpire: tc.expiring, Expires: now.Add(tc.credential)}

			expiration, err := uploadExpiration(request, creds, now)
			if !tc.valid {
				if !errors.Is(err, storage.ErrInvalid) || !expiration.IsZero() {
					t.Fatalf("invalid window admitted: %v", err)
				}

				return
			}

			deadline := request.ExpiresAt
			if tc.expiring && creds.Expires.Before(deadline) {
				deadline = creds.Expires
			}

			if err != nil || !expiration.Equal(deadline.Truncate(time.Second)) || expiration.After(request.ExpiresAt) || tc.expiring && expiration.After(creds.Expires) {
				t.Fatalf("effective deadline differs: %v", err)
			}
		})
	}
}

// TestSigningCapability rejects unsupported profiles and invalid input before credentials.
func TestSigningCapability(t *testing.T) {
	options := testOptions(t, "http://127.0.0.1:1")

	var calls atomic.Int32

	options.Credentials = aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		calls.Add(1)

		return aws.Credentials{AccessKeyID: "fixture-key", SecretAccessKey: "fixture-secret"}, nil
	})
	store := newTestStore(t, options)
	request := storage.UploadRequest{Key: "object", ContentType: "text/plain", MaxBytes: 8, ExpiresAt: time.Now().Add(time.Minute)}

	grant, err := store.SignUpload(context.Background(), request)
	if !errors.Is(err, storage.ErrUnsupported) || grant.Fields != nil || calls.Load() != 0 {
		t.Fatal("unsupported profile signed or performed I/O")
	}

	options.PostPolicy = PostPolicyProfile(255)

	_, err = NewStore(context.Background(), options)
	if !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("unknown profile admitted: %v", err)
	}

	options.PostPolicy = PostPolicyMinIO
	store = newTestStore(t, options)
	request.MaxBytes = 0

	_, err = store.SignUpload(context.Background(), request)
	if !errors.Is(err, storage.ErrInvalid) || calls.Load() != 0 {
		t.Fatal("invalid request reached credentials")
	}

	request.MaxBytes = 8
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.SignUpload(ctx, request)
	if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("cancelled request lost cause or reached credentials")
	}
}

// TestSigningCredentials binds temporary token fields and actual credential expiry.
func TestSigningCredentials(t *testing.T) {
	options := testOptions(t, "http://127.0.0.1:1")
	options.PostPolicy = PostPolicyMinIO
	expires := time.Now().Add(30 * time.Second)
	options.Credentials = aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "fixture-key", SecretAccessKey: "fixture-secret", SessionToken: "fixture-temporary-token", CanExpire: true, Expires: expires}, nil
	})
	store := newTestStore(t, options)
	request := storage.UploadRequest{Key: "object", ContentType: "text/plain", MaxBytes: 8, ExpiresAt: time.Now().Add(time.Minute)}

	grant, err := store.SignUpload(context.Background(), request)
	if err != nil || !grant.ExpiresAt.Equal(expires.UTC().Truncate(time.Second)) || grant.Fields["X-Amz-Security-Token"] == "" {
		t.Fatalf("credential deadline or required token differs: %v", err)
	}

	assertPolicy(t, grant, request, options.Bucket)

	for _, tc := range []struct {
		name   string
		change func(*aws.Credentials)
	}{
		{"missing key", func(c *aws.Credentials) { c.AccessKeyID = "" }},
		{"missing secret", func(c *aws.Credentials) { c.SecretAccessKey = "" }},
		{"long key", func(c *aws.Credentials) { c.AccessKeyID = strings.Repeat("x", 257) }},
		{"long secret", func(c *aws.Credentials) { c.SecretAccessKey = strings.Repeat("x", 257) }},
		{"long token", func(c *aws.Credentials) { c.SessionToken = strings.Repeat("x", (64<<10)+1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creds := aws.Credentials{AccessKeyID: "fixture-key", SecretAccessKey: "fixture-secret"}
			tc.change(&creds)

			expiration, err := uploadExpiration(request, creds, time.Now())
			if !errors.Is(err, storage.ErrInvalid) || !expiration.IsZero() {
				t.Fatal("invalid or unbounded provider signing material admitted")
			}
		})
	}
}
