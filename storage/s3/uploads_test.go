//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"hatmax.adrianpk.com/storage"
)

func liveSigner(t *testing.T) (*Store, Options) {
	t.Helper()
	options := testOptions(t, live.endpoint)
	options.Bucket = live.bucket
	options.Credentials = live.provider
	options.PostPolicy = PostPolicyMinIO

	return newTestStore(t, options), options
}

func signRequest(t *testing.T, store *Store, request storage.UploadRequest) storage.UploadGrant {
	t.Helper()

	grant, err := store.SignUpload(context.Background(), request)
	if err != nil {
		t.Fatal("sign fixture upload:", err)
	}

	assertPolicy(t, grant, request, live.bucket)

	return grant
}

func formRequest(ctx context.Context, grant storage.UploadGrant, payload io.Reader, size int64) (*http.Request, error) {
	var envelope bytes.Buffer

	writer := multipart.NewWriter(&envelope)
	keys := make([]string, 0, len(grant.Fields))

	for key := range grant.Fields {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		err := writer.WriteField(key, grant.Fields[key])
		if err != nil {
			return nil, err
		}
	}

	_, err := writer.CreateFormFile("file", "document.bin")
	if err != nil {
		return nil, err
	}

	headerSize := envelope.Len()

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// The file is the last form part. Its explicit length permits real streaming
	// while retaining the provider's required non-chunked multipart framing.
	framing := envelope.Bytes()
	body := io.MultiReader(bytes.NewReader(framing[:headerSize]), payload, bytes.NewReader(framing[headerSize:]))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, grant.URL, body)
	if err != nil {
		return nil, err
	}

	request.ContentLength = int64(len(framing)) + size
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return request, nil
}

func submitForm(t *testing.T, grant storage.UploadGrant, payload string) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	request, err := formRequest(ctx, grant, strings.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal("create fixture form:", err)
	}

	return sendForm(t, request)
}

func sendForm(t *testing.T, request *http.Request) int {
	t.Helper()

	client := *live.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	response, err := client.Do(request)
	if err != nil {
		t.Fatal("submit fixture form:", err)
	}

	// Provider diagnostics can contain sensitive form material. Retain status
	// only, bound the drain, and never print grant fields or response bodies.
	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))

	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("close fixture form response: %v/%v", readErr, closeErr)
	}

	return response.StatusCode
}

func readSignedObject(t *testing.T, store *Store, key string, maximum int64) (string, storage.Object) {
	t.Helper()

	reader, object, err := store.Open(context.Background(), key, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		t.Fatal(err)
	}

	data, readErr := io.ReadAll(reader)

	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("complete signed object read: %v/%v", readErr, closeErr)
	}

	return string(data), object
}

func scopeBucket(t *testing.T) string {
	t.Helper()

	name := "scope-" + strings.ToLower(rand.Text())

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	_, err := live.admin.CreateBucket(ctx, &sdk.CreateBucketInput{Bucket: aws.String(name)})
	if err != nil {
		t.Fatal(err)
	}

	live.record("scope-probe-bucket: created private\n")
	t.Cleanup(func() {
		cleanup, finish := context.WithTimeout(context.Background(), 120*time.Second)
		defer finish()

		// If the negative assertion fails, its exact owned target still gets removed.
		_, deleteErr := live.admin.DeleteObject(cleanup, &sdk.DeleteObjectInput{Bucket: aws.String(name), Key: aws.String("signed/exact")})

		_, bucketErr := live.admin.DeleteBucket(cleanup, &sdk.DeleteBucketInput{Bucket: aws.String(name)})
		if deleteErr != nil || bucketErr != nil {
			t.Errorf("scope-probe bucket cleanup: %v/%v", deleteErr, bucketErr)

			return
		}

		live.record("scope-probe-bucket: removed\n")
	})

	return name
}

// TestSignedUploads proves actual exact-target/type/size/signature enforcement and reuse.
func TestSignedUploads(t *testing.T) {
	store, options := liveSigner(t)
	request := storage.UploadRequest{Key: "signed/exact", ContentType: "text/plain", MinBytes: 2, MaxBytes: 8, ExpiresAt: time.Now().Add(time.Minute)}
	grant := signRequest(t, store, request)
	otherBucket := scopeBucket(t)

	for _, payload := range []string{"ab", "12345678", "reused"} {
		status := submitForm(t, grant, payload)
		if status < 200 || status >= 300 {
			t.Fatalf("valid or reused form rejected: status=%d", status)
		}

		data, object := readSignedObject(t, store, request.Key, request.MaxBytes)
		if data != payload || object.ContentType != request.ContentType || object.SizeBytes != int64(len(payload)) {
			t.Fatal("accepted form bytes or declared metadata differ")
		}
	}

	t.Run("empty allowed", func(t *testing.T) {
		emptyRequest := request
		emptyRequest.Key = "signed/empty"
		emptyRequest.MinBytes = 0
		empty := signRequest(t, store, emptyRequest)

		status := submitForm(t, empty, "")
		if status < 200 || status >= 300 {
			t.Fatalf("admitted empty file rejected: status=%d", status)
		}

		data, object := readSignedObject(t, store, emptyRequest.Key, emptyRequest.MaxBytes)
		if data != "" || object.SizeBytes != 0 {
			t.Fatal("empty form bytes differ")
		}
	})

	cases := []struct {
		name    string
		change  func(*storage.UploadGrant)
		payload string
	}{
		{"key", func(g *storage.UploadGrant) { g.Fields["key"] = "signed/other" }, "valid"},
		{"type", func(g *storage.UploadGrant) { g.Fields["Content-Type"] = "application/json" }, "valid"},
		{"bucket", func(g *storage.UploadGrant) { g.URL = live.endpoint + "/" + otherBucket }, "valid"},
		{"policy", func(g *storage.UploadGrant) {
			policy := decodePolicy(t, *g)
			policy.Expiration = policy.Expiration.Add(time.Hour)

			data, err := json.Marshal(policy)
			if err != nil {
				t.Fatal("encode altered fixture policy")
			}

			g.Fields["policy"] = base64.StdEncoding.EncodeToString(data)
		}, "valid"},
		{"signature", func(g *storage.UploadGrant) {
			prefix := "0"
			if g.Fields["X-Amz-Signature"][0] == '0' {
				prefix = "1"
			}

			g.Fields["X-Amz-Signature"] = prefix + g.Fields["X-Amz-Signature"][1:]
		}, "valid"},
		{"minimum", func(*storage.UploadGrant) {}, "x"},
		{"maximum", func(*storage.UploadGrant) {}, "123456789"},
		{"public ACL", func(g *storage.UploadGrant) { g.Fields["acl"] = "public-read" }, "valid"},
		{"redirect", func(g *storage.UploadGrant) {
			g.Fields["success_action_redirect"] = "https://redirect.example.test/complete"
		}, "valid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			altered := grant
			altered.Fields = maps.Clone(grant.Fields)
			tc.change(&altered)

			status := submitForm(t, altered, tc.payload)
			if status < 400 || status >= 500 {
				t.Fatalf("altered form not rejected by policy: status=%d", status)
			}

			data, _ := readSignedObject(t, store, request.Key, request.MaxBytes)
			if data != "reused" {
				t.Fatal("rejected form changed the previous complete object")
			}
		})
	}

	_, err := store.Stat(context.Background(), "signed/other")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("altered key escaped its target")
	}

	creds, err := live.provider.Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, value := range grant.Fields {
		if strings.Contains(value, creds.SecretAccessKey) {
			t.Fatal("grant exposed reusable secret signing material")
		}
	}

	policy, err := base64.StdEncoding.DecodeString(grant.Fields["policy"])
	if err != nil || strings.Contains(string(policy), creds.SecretAccessKey) {
		t.Fatal("policy exposed reusable secret signing material")
	}

	assertEmptySpool(t, options.SpoolDirectory)
}

// TestSigningExpiry checks real expiry and completion of a transfer started earlier.
func TestSigningExpiry(t *testing.T) {
	store, _ := liveSigner(t)
	request := storage.UploadRequest{Key: "expiry/object", ContentType: "text/plain", MinBytes: 2, MaxBytes: 8, ExpiresAt: time.Now().Add(3 * time.Second)}
	grant := signRequest(t, store, request)

	status := submitForm(t, grant, "valid")
	if status < 200 || status >= 300 {
		t.Fatalf("fresh form rejected: status=%d", status)
	}

	waitForExpiry(t, grant.ExpiresAt)

	status = submitForm(t, grant, "later")
	if status < 400 || status >= 500 {
		t.Fatalf("expired form admitted: status=%d", status)
	}

	data, _ := readSignedObject(t, store, request.Key, request.MaxBytes)
	if data != "valid" {
		t.Fatal("expired form changed accepted bytes")
	}

	t.Run("in flight", func(t *testing.T) { testInFlight(t, store) })
}

func waitForExpiry(t *testing.T, expiry time.Time) {
	t.Helper()

	delay := time.Until(expiry.Add(150 * time.Millisecond))
	if delay > 10*time.Second {
		t.Fatal("fixture expiry wait exceeded its finite bound")
	}

	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		<-timer.C
	}
}

// TestSnapshotCompletion validates one complete response and writes those same bytes.
func TestSnapshotCompletion(t *testing.T) {
	store, options := liveSigner(t)
	request := storage.UploadRequest{Key: "completion/staging", ContentType: "text/plain", MinBytes: 1, MaxBytes: 64, ExpiresAt: time.Now().Add(time.Minute)}
	grant := signRequest(t, store, request)
	original := "approved snapshot"

	status := submitForm(t, grant, original)
	if status < 200 || status >= 300 {
		t.Fatalf("staging form rejected: status=%d", status)
	}

	reader, observed, err := store.Open(context.Background(), request.Key, storage.ReadOptions{MaxBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	status = submitForm(t, grant, "staging overwritten")
	if status < 200 || status >= 300 {
		t.Fatalf("staging overwrite rejected: status=%d", status)
	}

	snapshot, readErr := io.ReadAll(reader)

	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || string(snapshot) != original || observed.SizeBytes != int64(len(snapshot)) || !strings.HasPrefix(string(snapshot), "approved ") {
		t.Fatal("complete opened snapshot validation failed")
	}

	finalKey := "completion/final"

	_, err = store.Put(context.Background(), finalKey, bytes.NewReader(snapshot), storage.WriteOptions{ContentType: observed.ContentType, MaxBytes: 64})
	if err != nil {
		t.Fatal(err)
	}

	status = submitForm(t, grant, "staging overwritten again")
	if status < 200 || status >= 300 {
		t.Fatalf("later grant reuse rejected: status=%d", status)
	}

	final, object := readSignedObject(t, store, finalKey, 64)
	if final != original || object.ContentType != observed.ContentType {
		t.Fatal("staging reuse changed the distinct finalized snapshot")
	}

	assertEmptySpool(t, options.SpoolDirectory)
}
