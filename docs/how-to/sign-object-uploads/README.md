<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Submit and Complete a Signed Object Upload

Start with [private S3 configuration](../store-private-objects/README.md).
Use an isolated owned bucket: this example owns and may replace
`guide/staging-document.txt` and `guide/final-document.txt`. Set the deployment's
`OBJECT_STORAGE_BUCKET`, `OBJECT_STORAGE_REGION`, `OBJECT_STORAGE_ENDPOINT` and
`OBJECT_STORAGE_SPOOL`; the spool directory already exists. Use the SDK credential
chain and verified HTTPS. The tested profile must match the deployment.

## Issue and submit the form

In an application, authenticate and admit the request before calling
`UploadSigner.SignUpload`. Choose a disposable staging key under application
control. Admit an exact declared content type, inclusive byte range and an
absolute expiry at most 15 minutes away. Do not print or persist grant fields.
The executable uses these fixed exercise keys, a 1 MiB maximum, and two minutes:

<!-- object-storage-source: examples/object-storage/remote/main.go#run -->
```go
func run(ctx context.Context) (result error) {
	store, err := newStore(ctx)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, store.Close()) }()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result = errors.Join(result, store.Delete(cleanup, stagingKey), store.Delete(cleanup, finalKey))
	}()

	grant, err := store.SignUpload(ctx, storage.UploadRequest{
		Key: stagingKey, ContentType: "text/plain", MinBytes: 1,
		MaxBytes: maximum, ExpiresAt: time.Now().Add(2 * time.Minute),
	})
	if err != nil {
		return err
	}

	err = submit(ctx, grant)
	if err != nil {
		return err
	}

	return complete(ctx, store)
}
```

`submit` copies every returned field unchanged, then appends one file part last.
The form field `Content-Type` is the declared object's type. The HTTP header is
the multipart boundary type. The fixed small demo document and bounded grant
fields fit a finite buffer; `bytes.Reader` supplies an explicit Content-Length.
A general streaming implementation must bound its own input and multipart size.
Use a finite HTTP deadline and reject redirects.

<!-- object-storage-source: examples/object-storage/remote/upload.go#submit -->
```go
func submit(ctx context.Context, grant storage.UploadGrant) error {
	// This example buffers one fixed small document and bounded signer fields.
	var body bytes.Buffer

	form := multipart.NewWriter(&body)
	fields := make([]string, 0, len(grant.Fields))
	for name := range grant.Fields {
		fields = append(fields, name)
	}

	slices.Sort(fields)

	for _, name := range fields {
		err := form.WriteField(name, grant.Fields[name])
		if err != nil {
			return err
		}
	}

	part, err := form.CreateFormFile("file", "document.txt")
	if err != nil {
		return err
	}

	_, err = io.WriteString(part, "Approved document\nPrivate bytes without image metadata.\n")
	if err != nil {
		return err
	}

	err = form.Close()
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, grant.URL, bytes.NewReader(body.Bytes()))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", form.FormDataContentType())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	err = errors.Join(readErr, response.Body.Close())
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("signed POST rejected: HTTP %d", response.StatusCode)
	}

	return nil
}
```

The provider enforces exact bucket/key/type, file-byte range and signed expiry.
A declared type is not format validation. The returned `ExpiresAt` is the actual
policy deadline, rounded down to whole seconds and shortened for known credential
expiry. A grant may be reused before expiry. An earlier transfer may finish after
expiry; expiry does not revoke it or change application membership.

## Validate one complete snapshot

Recheck current application authority when completing an upload. Open staging
once, read to successful bounded EOF, close it, validate those bytes and write
those same bytes to a distinct final key. Never use `Stat` followed by a later
unvalidated copy from staging. The complete example verifies its known document
and reads back final bytes:

<!-- object-storage-source: examples/object-storage/remote/completion.go#complete -->
```go
func complete(ctx context.Context, store storage.Store) error {
	reader, opened, err := store.Open(ctx, stagingKey, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		return err
	}

	// Only successful EOF establishes a complete bounded snapshot.
	snapshot, readErr := io.ReadAll(reader)
	err = errors.Join(readErr, reader.Close())
	if err != nil {
		return err
	}

	expected := []byte("Approved document\nPrivate bytes without image metadata.\n")
	if opened.ContentType != "text/plain" || opened.SizeBytes != int64(len(snapshot)) || !bytes.Equal(snapshot, expected) {
		return errors.New("document validation failed")
	}

	// The application rechecks current authority before this step in production.
	_, err = store.Put(ctx, finalKey, bytes.NewReader(snapshot), storage.WriteOptions{ContentType: "text/plain", MaxBytes: maximum})
	if err != nil {
		return err
	}

	final, _, err := store.Open(ctx, finalKey, storage.ReadOptions{MaxBytes: maximum})
	if err != nil {
		return err
	}

	accepted, readErr := io.ReadAll(final)
	err = errors.Join(readErr, final.Close())
	if err != nil {
		return err
	}

	if !bytes.Equal(accepted, snapshot) {
		return errors.New("final readback differs from validated snapshot")
	}

	return nil
}
```

The demo does not mark a database record available. Production completion owns
that durable transition, current permission checks and retry/reconciliation.
Its cleanup runs with a separate finite context and deletes its two owned keys
before closing the store. A failed or uncertain mutation is reconciled through
[the failure procedure](../reconcile-object-storage/README.md).

## Run and verify

From the repository root, after supplying the deployment configuration:

```sh
GOWORK=off go run ./examples/object-storage/remote
```

Expected output is `remote object workflow complete`. The command actually
submits the multipart form, validates the complete staging snapshot, verifies
identical final bytes, then deletes both exercise objects. It prints no grants,
credentials, object bodies or provider response bodies.

For an entirely owned native fixture instead of a deployment account, run:

```sh
GOWORK=off scripts/check-object-storage.sh slice 4
```

This checks the same published executable against the pinned actual server,
including exact accepted bytes and confirmed object/bucket/server/root cleanup.
Plaintext is enabled only for that literal-loopback fixture. Acceptance covers
the identified server profile. See [the reference](../../reference/storage/README.md)
and [bearer/snapshot ownership](../../explanation/private-object-ownership/README.md).
