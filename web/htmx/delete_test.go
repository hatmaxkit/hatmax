package htmx

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/hatmaxkit/hatmax/log"
)

func TestRespondDelete(t *testing.T) {
	logger := log.NewTestLogger("error")

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success",
			err:        nil,
			wantStatus: 200,
			wantBody:   "",
		},
		{
			name:       "error",
			err:        errors.New("delete failed"),
			wantStatus: 500,
			wantBody:   "Cannot delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			RespondDelete(w, tt.err, logger)

			if w.Code != tt.wantStatus {
				t.Errorf("RespondDelete() status = %v, want %v", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" && w.Body.String() != tt.wantBody+"\n" {
				t.Errorf("RespondDelete() body = %q, want %q", w.Body.String(), tt.wantBody+"\n")
			}

			if tt.wantBody == "" && w.Body.String() != "" {
				t.Errorf("RespondDelete() body = %q, want empty", w.Body.String())
			}
		})
	}
}
