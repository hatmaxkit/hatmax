package mailer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMailgunMailerSend(t *testing.T) {
	var (
		gotAuth string
		gotPath string
		gotBody string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mailer := NewMailgunMailer(MailgunConfig{
		Config: Config{
			DefaultFrom: Address{Email: "noreply@example.com"},
		},
		APIKey:  "mg-key",
		Domain:  "mg.example.com",
		BaseURL: server.URL,
	})
	mailer.client = server.Client()

	err := mailer.Send(context.Background(), &Message{
		From:    Address{Email: "from@example.com"},
		To:      []Address{{Email: "a@example.com"}},
		Subject: "Weekly",
		Text:    "body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotPath != "/v3/mg.example.com/messages" {
		t.Fatalf("unexpected path: %s", gotPath)
	}

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("expected basic auth header")
	}

	if !strings.Contains(gotBody, "subject=Weekly") {
		t.Fatalf("payload should contain subject")
	}
}

func TestMailgunMailerValidation(t *testing.T) {
	m := NewMailgunMailer(MailgunConfig{
		Config: Config{DefaultFrom: Address{Email: "noreply@example.com"}},
	})

	err := m.Send(context.Background(), &Message{
		From:    Address{Email: "from@example.com"},
		To:      []Address{{Email: "a@example.com"}},
		Subject: "test",
		Text:    "body",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
