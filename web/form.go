package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FormValues struct {
	r *http.Request
}

func ParseForm(r *http.Request) (*FormValues, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return &FormValues{r: r}, nil
}

func (f *FormValues) String(name string) string {
	return f.r.FormValue(name)
}

func (f *FormValues) StringOr(name, def string) string {
	v := f.r.FormValue(name)
	if v == "" {
		return def
	}
	return v
}

func (f *FormValues) Bool(name string) bool {
	v := f.r.FormValue(name)
	return v == "true" || v == "on"
}

func (f *FormValues) UUID(name string) (uuid.UUID, error) {
	return uuid.Parse(f.r.FormValue(name))
}

func ParseIDParam(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}
