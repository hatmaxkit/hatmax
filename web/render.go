package web

import (
	"html/template"
	"net/http"

	"hatmax.adrianpk.com/htmx"
	"hatmax.adrianpk.com/log"
)

func RenderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data interface{}, log log.Logger) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Errorf("Template render error (%s): %v", name, err)
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
	}
}

func RenderPartial(w http.ResponseWriter, tmpl *template.Template, name string, data interface{}, log log.Logger) {
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Errorf("Partial render error (%s): %v", name, err)
		http.Error(w, "Partial rendering error", http.StatusInternalServerError)
	}
}

func RedirectOrHXRedirect(w http.ResponseWriter, r *http.Request, url string) {
	if htmx.IsHTMXRequest(r) {
		htmx.Redirect(w, url)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}
