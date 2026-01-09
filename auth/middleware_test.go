package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
)

func TestRequireAuth(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user and session
	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "password123")

	// Handler that checks if user is in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r.Context())
		if !ok {
			t.Error("RequireAuth() user not in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if user.Email != "test@example.com" {
			t.Errorf("RequireAuth() user.Email = %v, want test@example.com", user.Email)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequireAuth(svc)
	wrappedHandler := middleware(handler)

	tests := []struct {
		name           string
		sessionCookie  *http.Cookie
		wantStatusCode int
		wantRedirect   bool
	}{
		{
			name: "valid session",
			sessionCookie: &http.Cookie{
				Name:  SessionCookieName,
				Value: session.Token,
			},
			wantStatusCode: http.StatusOK,
			wantRedirect:   false,
		},
		{
			name:           "no session cookie",
			sessionCookie:  nil,
			wantStatusCode: http.StatusSeeOther,
			wantRedirect:   true,
		},
		{
			name: "invalid session token",
			sessionCookie: &http.Cookie{
				Name:  SessionCookieName,
				Value: "invalid-token",
			},
			wantStatusCode: http.StatusSeeOther,
			wantRedirect:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.sessionCookie != nil {
				req.AddCookie(tt.sessionCookie)
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("RequireAuth() status = %v, want %v", w.Code, tt.wantStatusCode)
			}

			if tt.wantRedirect {
				location := w.Header().Get("Location")
				if location != "/signin" {
					t.Errorf("RequireAuth() redirect location = %v, want /signin", location)
				}
			}
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	queries := newMockQueries()
	cfg := config.New()
	logger := log.NewTestLogger("error")
	svc := NewService(queries, cfg, logger)

	// Create a test user and session
	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Signin(context.Background(), "test@example.com", "password123")

	// Handler that checks if user is in context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r.Context())
		if ok && user.Email != "test@example.com" {
			t.Errorf("OptionalAuth() user.Email = %v, want test@example.com", user.Email)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := OptionalAuth(svc)
	_ = middleware(handler)

	tests := []struct {
		name          string
		sessionCookie *http.Cookie
		wantUserInCtx bool
	}{
		{
			name: "valid session",
			sessionCookie: &http.Cookie{
				Name:  SessionCookieName,
				Value: session.Token,
			},
			wantUserInCtx: true,
		},
		{
			name:          "no session cookie",
			sessionCookie: nil,
			wantUserInCtx: false,
		},
		{
			name: "invalid session token",
			sessionCookie: &http.Cookie{
				Name:  SessionCookieName,
				Value: "invalid-token",
			},
			wantUserInCtx: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			if tt.sessionCookie != nil {
				req.AddCookie(tt.sessionCookie)
			}

			// Capture the context in the handler
			var capturedCtx context.Context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedCtx = r.Context()
				w.WriteHeader(http.StatusOK)
			})
			testWrapped := middleware(testHandler)

			w := httptest.NewRecorder()
			testWrapped.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("OptionalAuth() status = %v, want %v", w.Code, http.StatusOK)
			}

			_, ok := GetUser(capturedCtx)
			if ok != tt.wantUserInCtx {
				t.Errorf("OptionalAuth() user in context = %v, want %v", ok, tt.wantUserInCtx)
			}
		})
	}
}

func TestSetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	token := "test-token"
	maxAge := 3600

	SetSessionCookie(w, token, maxAge)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("SetSessionCookie() set %d cookies, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Errorf("SetSessionCookie() cookie.Name = %v, want %v", cookie.Name, SessionCookieName)
	}
	if cookie.Value != token {
		t.Errorf("SetSessionCookie() cookie.Value = %v, want %v", cookie.Value, token)
	}
	if cookie.MaxAge != maxAge {
		t.Errorf("SetSessionCookie() cookie.MaxAge = %v, want %v", cookie.MaxAge, maxAge)
	}
	if !cookie.HttpOnly {
		t.Error("SetSessionCookie() cookie.HttpOnly = false, want true")
	}
	if !cookie.Secure {
		t.Error("SetSessionCookie() cookie.Secure = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SetSessionCookie() cookie.SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()

	ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("ClearSessionCookie() set %d cookies, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Errorf("ClearSessionCookie() cookie.Name = %v, want %v", cookie.Name, SessionCookieName)
	}
	if cookie.Value != "" {
		t.Errorf("ClearSessionCookie() cookie.Value = %v, want empty", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("ClearSessionCookie() cookie.MaxAge = %v, want -1", cookie.MaxAge)
	}
}
