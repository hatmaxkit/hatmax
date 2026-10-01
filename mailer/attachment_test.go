// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type mailgunCapture struct {
	fields      url.Values
	attachments []Attachment
	contentType string
	user        string
	key         string
	path        string
	err         error
}

// Capture the real HTTP encoding rather than an internal builder: accepted
// requests must contain every attachment and keep all existing message fields.
func TestMailgunAttachments(t *testing.T) {
	for _, tc := range []struct {
		name        string
		attachments []Attachment
	}{
		{name: "no attachments"},
		{name: "empty slice", attachments: []Attachment{}},
		{name: "text", attachments: []Attachment{{Filename: "invoice.txt", ContentType: "text/plain", Data: []byte("invoice\n")}}},
		{name: "binary", attachments: []Attachment{{Filename: "invoice.bin", ContentType: "application/octet-stream", Data: []byte{0, 1, 255, '\r', '\n'}}}},
		{name: "default MIME", attachments: []Attachment{{Filename: "invoice.dat", Data: []byte("invoice")}}},
		{name: "parameterized MIME", attachments: []Attachment{{Filename: "invoice.txt", ContentType: "text/plain; charset=utf-8", Data: []byte("invoice")}}},
		{name: "escaped filename", attachments: []Attachment{{Filename: "invoice \"draft\".txt", ContentType: "text/plain", Data: []byte("invoice")}}},
		{name: "Unicode filename", attachments: []Attachment{{Filename: "invoice-\u00e9.txt", ContentType: "text/plain", Data: []byte("invoice")}}},
		{name: "empty data", attachments: []Attachment{{Filename: "empty.txt", ContentType: "text/plain", Data: []byte{}}}},
		{name: "nil data", attachments: []Attachment{{Filename: "empty.txt", ContentType: "text/plain"}}},
		{name: "multiple", attachments: []Attachment{
			{Filename: "invoice.txt", ContentType: "text/plain", Data: []byte("first")},
			{Filename: "invoice.txt", ContentType: "application/octet-stream", Data: []byte{0, 255}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			captures := make(chan mailgunCapture, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captures <- captureMailgun(r)

				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)

			m := NewMailgunMailer(MailgunConfig{
				Config: Config{DefaultFrom: Address{Email: "default@example.com", Name: "Default"}},
				APIKey: "test-key", Domain: "example.com", BaseURL: server.URL,
			})
			msg := senderMessage(Address{}, false, "")
			msg.Attachments = tc.attachments
			msg.CC = append(msg.CC, Address{Email: "second-copy@example.com"})
			msg.BCC = append(msg.BCC, Address{Email: "second-blind@example.com"})
			msg.Headers = map[string]string{"X-Notice": "invoice", "X-Label": "a&b=1"}

			before, err := json.Marshal(msg)
			if err != nil {
				t.Fatal(err)
			}

			err = m.Send(context.Background(), msg)
			if err != nil {
				t.Fatalf("Send: %v", err)
			}

			got := <-captures
			if got.err != nil {
				t.Fatal(got.err)
			}

			wantType := "application/x-www-form-urlencoded"
			if len(tc.attachments) > 0 {
				wantType = "multipart/form-data"
			}

			if got.contentType != wantType || got.user != "api" || got.key != "test-key" || got.path != "/v3/example.com/messages" {
				t.Fatalf("request metadata: %+v", got)
			}

			wantFields := url.Values{
				"from": {m.cfg.DefaultFrom.String()}, "to": {"recipient@example.com"}, "subject": {"Notice"},
				"text": {"Text body"}, "html": {"<p>HTML body</p>"},
				"cc":         {"copy@example.com", "second-copy@example.com"},
				"bcc":        {"blind@example.com", "second-blind@example.com"},
				"h:Reply-To": {"reply@example.com"}, "h:X-Notice": {"invoice"}, "h:X-Label": {"a&b=1"},
			}
			if !reflect.DeepEqual(got.fields, wantFields) {
				t.Fatalf("fields: got %v; want %v", got.fields, wantFields)
			}

			if len(got.attachments) != len(tc.attachments) {
				t.Fatalf("got %d attachments; want %d", len(got.attachments), len(tc.attachments))
			}

			for i, want := range tc.attachments {
				wantMIME := want.ContentType
				if wantMIME == "" {
					wantMIME = "application/octet-stream"
				}

				attachment := got.attachments[i]
				if attachment.Filename != want.Filename || attachment.ContentType != wantMIME || !bytes.Equal(attachment.Data, want.Data) {
					t.Fatalf("attachment %d: got %+v; want %+v", i, attachment, want)
				}
			}

			after, err := json.Marshal(msg)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(before, after) {
				t.Fatal("Send changed the caller's message")
			}
		})
	}
}

func captureMailgun(r *http.Request) mailgunCapture {
	got := mailgunCapture{fields: url.Values{}, path: r.URL.Path}
	got.user, got.key, _ = r.BasicAuth()

	got.contentType, _, got.err = mime.ParseMediaType(r.Header.Get("Content-Type"))
	if got.err != nil {
		return got
	}

	if got.contentType != "multipart/form-data" {
		got.err = r.ParseForm()
		got.fields = r.PostForm

		return got
	}

	reader, err := r.MultipartReader()
	if err != nil {
		got.err = err

		return got
	}

	for {
		part, readErr := reader.NextPart()
		if errors.Is(readErr, io.EOF) {
			return got
		}

		if readErr != nil {
			got.err = readErr

			return got
		}

		data, dataErr := io.ReadAll(part)
		part.Close()

		if dataErr != nil {
			got.err = dataErr

			return got
		}

		if part.FormName() != "attachment" {
			got.fields.Add(part.FormName(), string(data))

			continue
		}

		_, params, parseErr := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if parseErr != nil {
			got.err = parseErr

			return got
		}

		got.attachments = append(got.attachments, Attachment{Filename: params["filename"], ContentType: part.Header.Get("Content-Type"), Data: data})
	}
}

// Unserializable attachment metadata must fail before a request can be accepted
// as success. Empty files are valid; absent filenames and malformed MIME are not.
func TestMailgunBadAttachment(t *testing.T) {
	for _, tc := range []struct {
		name       string
		attachment Attachment
		prior      bool
		wantErr    string
	}{
		{name: "missing filename", attachment: Attachment{Data: []byte("invoice")}, wantErr: "filename is required"},
		{name: "invalid MIME", attachment: Attachment{Filename: "invoice.txt", ContentType: "not a MIME type"}, wantErr: "invalid content type"},
		{name: "incomplete MIME", attachment: Attachment{Filename: "invoice.txt", ContentType: "text"}, wantErr: "invalid content type"},
		{name: "injected MIME", attachment: Attachment{Filename: "invoice.txt", ContentType: "text/plain\r\nX-Injected: value"}, wantErr: "invalid content type"},
		{name: "later missing filename", prior: true, wantErr: "filename is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- struct{}{}

				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)

			m := NewMailgunMailer(MailgunConfig{APIKey: "test-key", Domain: "example.com", BaseURL: server.URL})
			msg := senderMessage(Address{Email: "from@example.com"}, false, "")

			msg.Attachments = []Attachment{tc.attachment}
			if tc.prior {
				msg.Attachments = append([]Attachment{{Filename: "valid.txt", Data: []byte("valid")}}, msg.Attachments...)
			}

			err := m.Send(context.Background(), msg)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v; want %q", err, tc.wantErr)
			}

			select {
			case <-requests:
				t.Fatal("invalid attachment reached transport")
			default:
			}
		})
	}
}

// Adding attachments must not hide remote rejection or caller cancellation.
func TestMailgunAttachmentFailure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		canceled bool
	}{
		{name: "rejected", status: http.StatusBadRequest},
		{name: "canceled", status: http.StatusOK, canceled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				fmt.Fprint(w, "rejected message")
			}))
			t.Cleanup(server.Close)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tc.canceled {
				cancel()
			}

			m := NewMailgunMailer(MailgunConfig{APIKey: "test-key", Domain: "example.com", BaseURL: server.URL})
			msg := senderMessage(Address{Email: "from@example.com"}, false, "")
			msg.Attachments = []Attachment{{Filename: "invoice.txt", ContentType: "text/plain", Data: []byte("invoice")}}

			err := m.Send(ctx, msg)
			if tc.canceled {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v; want cancellation", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "unexpected status 400: rejected message") {
				t.Fatalf("got %v; want remote rejection", err)
			}
		})
	}
}
