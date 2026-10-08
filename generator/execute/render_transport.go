// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"
	"html"
	"strings"
)

var transportRenderers = map[string]recipeRenderer{
	"server_rendered_crud.migration":     renderMigration,
	"server_rendered_crud.queries":       renderQueries,
	"server_rendered_crud.handler":       renderHandler,
	"server_rendered_crud.page_template": renderPageTemplate,
	"server_rendered_crud.form_template": renderFormTemplate,
	"server_rendered_crud.row_template":  renderRowTemplate,
	"server_rendered_crud.handler_tests": renderHandlerTests,
}

func renderMigration(context renderContext, _ Edit) ([]byte, error) {
	var source strings.Builder
	source.WriteString("-- +migrate Up\n")
	fmt.Fprintf(&source, "CREATE TABLE %s (\n", context.table)
	source.WriteString("    id TEXT PRIMARY KEY,\n")

	for _, field := range context.fields {
		fmt.Fprintf(&source, "    %s %s NOT NULL,\n", field.Name, field.SQLType)
	}

	source.WriteString("    created_at TIMESTAMPTZ NOT NULL,\n    updated_at TIMESTAMPTZ NOT NULL\n);\n\n")
	fmt.Fprintf(&source, "CREATE INDEX %s_updated_at_idx ON %s (updated_at DESC, id DESC);\n\n", context.table, context.table)
	source.WriteString("-- +migrate Down\n")
	fmt.Fprintf(&source, "DROP TABLE %s;\n", context.table)

	return []byte(source.String()), nil
}

func renderQueries(context renderContext, _ Edit) ([]byte, error) {
	columns := renderSQLColumns(context)
	arguments := renderSQLArguments(context.fields)
	assignments := renderSQLAssignments(context.fields)

	// Selecting the whole relation keeps SQLC's table row type stable when
	// later migrations append fields after the original audit columns.
	return []byte(fmt.Sprintf(`-- name: List%s :many
SELECT *
FROM %s
ORDER BY created_at DESC, id DESC;

-- name: Get%s :one
SELECT *
FROM %s
WHERE id = sqlc.arg(id);

-- name: Create%s :exec
INSERT INTO %s (%s)
VALUES (%s);

-- name: Update%s :execrows
UPDATE %s
SET %s,
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: Delete%s :execrows
DELETE FROM %s
WHERE id = sqlc.arg(id);
`, context.plural, context.table, context.entity, context.table, context.entity, context.table, columns, arguments, context.entity, context.table, assignments, context.entity, context.table)), nil
}

func renderSQLColumns(context renderContext) string {
	values := []string{"id"}
	for _, field := range context.fields {
		values = append(values, field.Name)
	}

	values = append(values, "created_at", "updated_at")

	return strings.Join(values, ", ")
}

func renderSQLArguments(fields []renderField) string {
	values := []string{"sqlc.arg(id)"}
	for _, field := range fields {
		values = append(values, "sqlc.arg("+field.Name+")")
	}

	values = append(values, "sqlc.arg(created_at)", "sqlc.arg(updated_at)")

	return strings.Join(values, ", ")
}

func renderSQLAssignments(fields []renderField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, field.Name+" = sqlc.arg("+field.Name+")")
	}

	return strings.Join(values, ",\n    ")
}

func renderHandler(context renderContext, edit Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\n", context.feature)
	source.WriteString("import (\n\t\"context\"\n\t\"errors\"\n\t\"net/http\"\n")

	if fieldsHaveKind(context.fields, "integer") {
		source.WriteString("\t\"strconv\"\n")
	}

	if fieldsNeedTime(context.fields) {
		source.WriteString("\t\"time\"\n")
	}

	source.WriteString("\n")
	source.WriteString("\t\"github.com/go-chi/chi/v5\"\n\t\"hatmax.adrianpk.com/htmx\"\n\t\"hatmax.adrianpk.com/log\"\n")

	if fieldsNeedNormalization(context.fields) {
		source.WriteString("\t\"hatmax.adrianpk.com/validation\"\n")
	}

	source.WriteString("\t\"hatmax.adrianpk.com/web\"\n)\n\n")
	fmt.Fprintf(&source, "// FeatureService is the handler-owned %s workflow boundary.\ntype FeatureService interface {\n", context.feature)
	fmt.Fprintf(&source, "\tList(context.Context) ([]%s, error)\n", context.entity)
	fmt.Fprintf(&source, "\tCreate(context.Context, %sInput) (*%s, error)\n", context.entity, context.entity)
	fmt.Fprintf(&source, "\tUpdate(context.Context, string, %sInput) (*%s, error)\n", context.entity, context.entity)
	source.WriteString("\tDelete(context.Context, string) error\n}\n\n")
	source.WriteString("// TemplateRenderer is the handler-owned rendering boundary.\ntype TemplateRenderer interface {\n\tRender(http.ResponseWriter, string, string, interface{})\n\tRenderPartial(http.ResponseWriter, string, string, interface{})\n}\n\n")
	fmt.Fprintf(&source, "type PageView struct {\n\tTitle string\n\tValues []%s\n\tForm FormView\n}\n\n", context.entity)
	source.WriteString("type FormView struct {\n\tValues map[string]string\n\tErrors web.FormErrors\n}\n\n")
	source.WriteString("type Handler struct {\n\tservice FeatureService\n\ttemplates TemplateRenderer\n\tlog log.Logger\n}\n\n")
	source.WriteString("func NewHandler(service FeatureService, templates TemplateRenderer, logger log.Logger) *Handler {\n\treturn &Handler{service: service, templates: templates, log: logger}\n}\n\n")

	itemRoute := context.route + "/{id}"
	fmt.Fprintf(&source, "func (handler *Handler) RegisterRoutes(router chi.Router) {\n\trouter.Get(%q, handler.page)\n\trouter.Post(%q, handler.create)\n\trouter.Put(%q, handler.update)\n\trouter.Delete(%q, handler.delete)\n}\n\n", context.route, context.route, itemRoute, itemRoute)
	renderPageHandler(&source, context)
	renderCreateHandler(&source, context)
	renderUpdateHandler(&source, context)
	renderDeleteHandler(&source, context)
	renderInputParser(&source, context)
	renderHandlerHelpers(&source, context)

	return formatGo(edit.Target, source.String())
}

func renderPageHandler(source *strings.Builder, context renderContext) {
	source.WriteString("func (handler *Handler) page(response http.ResponseWriter, request *http.Request) {\n\tvalues, err := handler.service.List(request.Context())\n\tif err != nil {\n\t\thandler.internalError(response, \"list records\", err)\n\n\t\treturn\n\t}\n\n")
	fmt.Fprintf(source, "\thandler.templates.Render(response, %q, \"page\", PageView{Title: %q, Values: values, Form: emptyForm()})\n}\n\n", context.feature, context.label)
}

func renderCreateHandler(source *strings.Builder, context renderContext) {
	source.WriteString("func (handler *Handler) create(response http.ResponseWriter, request *http.Request) {\n\tinput, form := parseInput(request)\n\tif form.Errors.Any() {\n\t\thandler.renderInvalid(response, request, form)\n\n\t\treturn\n\t}\n\n\tvalue, err := handler.service.Create(request.Context(), input)\n\tif err != nil {\n\t\tif formErrors := web.FormErrorsFrom(err, \"Review the highlighted fields.\"); len(formErrors.Fields) > 0 {\n\t\t\tform.Errors = formErrors\n\t\t\thandler.renderInvalid(response, request, form)\n\n\t\t\treturn\n\t\t}\n\n\t\thandler.internalError(response, \"create record\", err)\n\n\t\treturn\n\t}\n\n\tif htmx.IsHTMXRequest(request) {\n")
	fmt.Fprintf(source, "\t\thandler.templates.RenderPartial(response, %q, \"row\", value)\n\n\t\treturn\n\t}\n\n\thttp.Redirect(response, request, %q, http.StatusSeeOther)\n}\n\n", context.feature, context.route)
}

func renderUpdateHandler(source *strings.Builder, context renderContext) {
	source.WriteString("func (handler *Handler) update(response http.ResponseWriter, request *http.Request) {\n\tinput, form := parseInput(request)\n\tif form.Errors.Any() {\n\t\thandler.renderInvalid(response, request, form)\n\n\t\treturn\n\t}\n\n\tvalue, err := handler.service.Update(request.Context(), chi.URLParam(request, \"id\"), input)\n\tif errors.Is(err, ErrNotFound) {\n\t\thttp.Error(response, \"Record not found\", http.StatusNotFound)\n\n\t\treturn\n\t}\n\n\tif err != nil {\n\t\tformErrors := web.FormErrorsFrom(err, \"Review the highlighted fields.\")\n\t\tif len(formErrors.Fields) > 0 {\n\t\t\tform.Errors = formErrors\n\t\t\thandler.renderInvalid(response, request, form)\n\n\t\t\treturn\n\t\t}\n\n\t\thandler.internalError(response, \"update record\", err)\n\n\t\treturn\n\t}\n\n\tif htmx.IsHTMXRequest(request) {\n")
	fmt.Fprintf(source, "\t\thandler.templates.RenderPartial(response, %q, \"row\", value)\n\n\t\treturn\n\t}\n\n\thttp.Redirect(response, request, %q, http.StatusSeeOther)\n}\n\n", context.feature, context.route)
}

func renderDeleteHandler(source *strings.Builder, context renderContext) {
	source.WriteString("func (handler *Handler) delete(response http.ResponseWriter, request *http.Request) {\n\terr := handler.service.Delete(request.Context(), chi.URLParam(request, \"id\"))\n\tif errors.Is(err, ErrNotFound) {\n\t\thttp.Error(response, \"Record not found\", http.StatusNotFound)\n\n\t\treturn\n\t}\n\n\tif err != nil {\n\t\thandler.internalError(response, \"delete record\", err)\n\n\t\treturn\n\t}\n\n\tif htmx.IsHTMXRequest(request) {\n\t\thtmx.RespondDelete(response)\n\n\t\treturn\n\t}\n\n")
	fmt.Fprintf(source, "\thttp.Redirect(response, request, %q, http.StatusSeeOther)\n}\n\n", context.route)
}

func renderInputParser(source *strings.Builder, context renderContext) {
	fmt.Fprintf(source, "func parseInput(request *http.Request) (%sInput, FormView) {\n\tform := emptyForm()\n\n\tvalues, err := web.ParseForm(request)\n\tif err != nil {\n\t\tform.Errors = web.NewFormErrors(\"The form could not be read.\")\n\n\t\treturn %sInput{}, form\n\t}\n\n", context.entity, context.entity)
	fmt.Fprintf(source, "\tinput := %sInput{}\n", context.entity)

	for _, field := range context.fields {
		renderParsedField(source, field)
	}

	source.WriteString("\n\treturn input, form\n}\n\n")
}

func renderParsedField(source *strings.Builder, field renderField) {
	fmt.Fprintf(source, "\n\tform.Values[%q] = values.String(%q)\n", field.Name, field.Name)

	switch field.ParseKind {
	case "boolean":
		fmt.Fprintf(source, "\tinput.%s = values.Bool(%q)\n", field.GoName, field.Name)
	case "integer":
		fmt.Fprintf(source, "\tif form.Values[%q] != \"\" {\n\t\tparsed, parseErr := strconv.ParseInt(form.Values[%q], 10, 64)\n\t\tif parseErr != nil {\n\t\t\tform.Errors.Add(%q, \"must be an integer\")\n\t\t} else {\n\t\t\tinput.%s = parsed\n\t\t}\n\t}\n", field.Name, field.Name, field.Name, field.GoName)
	case "date":
		fmt.Fprintf(source, "\tif form.Values[%q] != \"\" {\n\t\tparsed, parseErr := time.Parse(\"2006-01-02\", form.Values[%q])\n\t\tif parseErr != nil {\n\t\t\tform.Errors.Add(%q, \"must be a date\")\n\t\t} else {\n\t\t\tinput.%s = parsed\n\t\t}\n\t}\n", field.Name, field.Name, field.Name, field.GoName)
	case "timestamp":
		fmt.Fprintf(source, "\tif form.Values[%q] != \"\" {\n\t\tparsed, parseErr := time.Parse(\"2006-01-02T15:04\", form.Values[%q])\n\t\tif parseErr != nil {\n\t\t\tform.Errors.Add(%q, \"must be a date and time\")\n\t\t} else {\n\t\t\tinput.%s = parsed\n\t\t}\n\t}\n", field.Name, field.Name, field.Name, field.GoName)
	default:
		fmt.Fprintf(source, "\tinput.%s = validation.NormalizeText(form.Values[%q])\n", field.GoName, field.Name)
	}
}

func renderHandlerHelpers(source *strings.Builder, context renderContext) {
	source.WriteString("func emptyForm() FormView {\n\treturn FormView{Values: make(map[string]string), Errors: web.NewFormErrors(\"\")}\n}\n\n")
	source.WriteString("func (handler *Handler) renderInvalid(response http.ResponseWriter, request *http.Request, form FormView) {\n\tresponse.WriteHeader(http.StatusUnprocessableEntity)\n\n\tif htmx.IsHTMXRequest(request) {\n")
	fmt.Fprintf(source, "\t\thandler.templates.RenderPartial(response, %q, \"form\", form)\n\n\t\treturn\n\t}\n\n\tvalues, err := handler.service.List(request.Context())\n\tif err != nil {\n\t\thandler.log.Errorf(\"list records after invalid form: %%v\", err)\n\t}\n\n\thandler.templates.Render(response, %q, \"page\", PageView{Title: %q, Values: values, Form: form})\n}\n\n", context.feature, context.feature, context.label)
	source.WriteString("func (handler *Handler) internalError(response http.ResponseWriter, operation string, err error) {\n\thandler.log.Errorf(\"%s: %v\", operation, err)\n\thttp.Error(response, \"The request could not be completed.\", http.StatusInternalServerError)\n}\n")
}

func renderPageTemplate(context renderContext, _ Edit) ([]byte, error) {
	return []byte(fmt.Sprintf(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Title}}</title></head>
<body>
<main>
  <h1>{{.Title}}</h1>
  <section id="%s-form">{{template "assets/templates/%s/form.html" .Form}}</section>
  <section id="%s-list">
    {{range .Values}}{{template "assets/templates/%s/row.html" .}}{{else}}<p>No records yet.</p>{{end}}
  </section>
</main>
</body>
</html>
`, context.feature, context.feature, context.feature, context.feature)), nil
}

func renderFormTemplate(context renderContext, _ Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, `<form method="post" action="%s" {{ hxAttrs (hxPost %q) (hxTargetID %q) hxSwapAfterBegin }}>
`, context.route, context.route, context.feature+"-list")
	source.WriteString("  {{if .Errors.General}}<p role=\"alert\">{{.Errors.General}}</p>{{end}}\n")

	for _, field := range context.fields {
		fmt.Fprintf(&source, "  <label>%s\n", field.Label)

		if field.HTMLType == "checkbox" {
			fmt.Fprintf(&source, "    <input type=\"checkbox\" name=\"%s\" value=\"true\">\n", field.Name)
		} else {
			fmt.Fprintf(&source, "    <input type=\"%s\" name=\"%s\" value=\"{{index .Values %q}}\"%s>\n", field.HTMLType, field.Name, field.Name, validationAttributes(context, field))
		}

		fmt.Fprintf(&source, "    {{with .Errors.First %q}}<span role=\"alert\">{{.}}</span>{{end}}\n  </label>\n", field.Name)
	}

	source.WriteString("  <button type=\"submit\">Create</button>\n</form>\n")

	return []byte(source.String()), nil
}

func renderRowTemplate(context renderContext, _ Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, "<article id=\"%s-{{.ID}}\">\n  <form method=\"post\" action=\"%s/{{.ID}}\" {{ hxAttrs (hxPut (printf %q .ID)) (hxTargetID (printf %q .ID)) hxSwapOuter }}>", context.feature, context.route, context.route+"/%s", context.feature+"-%s")

	for _, field := range context.fields {
		if field.HTMLType == "checkbox" {
			fmt.Fprintf(&source, "\n    <label>%s <input type=\"checkbox\" name=\"%s\" value=\"true\" {{if .%s}}checked{{end}}></label>", field.Label, field.Name, field.GoName)
		} else {
			fmt.Fprintf(&source, "\n    <label>%s <input type=\"%s\" name=\"%s\" value=\"{{.%s}}\"%s></label>", field.Label, field.HTMLType, field.Name, field.GoName, validationAttributes(context, field))
		}
	}

	source.WriteString("\n    <button type=\"submit\">Save</button>\n  </form>\n")
	fmt.Fprintf(&source, "  <button type=\"button\" {{ hxAttrs (hxDelete (printf %q .ID)) (hxTargetID (printf %q .ID)) hxSwapDelete }}>Delete</button>\n</article>\n", context.route+"/%s", context.feature+"-%s")

	return []byte(source.String()), nil
}

func validationAttributes(context renderContext, field renderField) string {
	attributes := ""
	if field.Required || (context.validation != nil && context.validation.Field == field.Name && context.validation.Kind == "required") {
		attributes += " required"
	}

	if context.validation == nil || context.validation.Field != field.Name {
		return attributes
	}

	switch context.validation.Kind {
	case "minimum":
		attributes += " min=\"" + context.validation.Value + "\""
	case "maximum":
		attributes += " max=\"" + context.validation.Value + "\""
	case "min_length":
		attributes += " minlength=\"" + context.validation.Value + "\""
	case "max_length":
		attributes += " maxlength=\"" + context.validation.Value + "\""
	case "pattern":
		attributes += " pattern=\"" + html.EscapeString(context.validation.Value) + "\""
	}

	return attributes
}

func renderHandlerTests(context renderContext, edit Edit) ([]byte, error) {
	validForm := renderFormValues(context.fields)
	validationImport := ""
	validationTest := ""

	if context.validation != nil {
		validationImport = "\n\t\"hatmax.adrianpk.com/validation\""
		validationTest = fmt.Sprintf(`

func TestHandlerMapsValidationErrors(t *testing.T) {
	service := &handlerService{createErr: validation.NewSingleError(%q, %q)}
	renderer := &renderedTemplate{}
	handler := NewHandler(service, renderer, log.NewNoopLogger())
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, %q, strings.NewReader(%q))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity || renderer.name != "form" || !renderer.partial {
		t.Fatalf("validation response = %%d, %%#v", response.Code, renderer)
	}
}
`, context.validation.Field, validationMessage(*context.validation), context.route, validForm)
	}

	source := fmt.Sprintf(`package %s

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/log"%s
)

type handlerService struct {
	created bool
	createErr error
}

func (service *handlerService) List(context.Context) ([]%s, error) {
	return nil, nil
}

func (service *handlerService) Create(_ context.Context, _ %sInput) (*%s, error) {
	if service.createErr != nil {
		return nil, service.createErr
	}

	service.created = true

	return &%s{ID: "created"}, nil
}

func (service *handlerService) Update(context.Context, string, %sInput) (*%s, error) {
	return &%s{ID: "updated"}, nil
}

func (service *handlerService) Delete(context.Context, string) error {
	return nil
}

type renderedTemplate struct {
	name string
	partial bool
}

func (renderer *renderedTemplate) Render(_ http.ResponseWriter, _, name string, _ interface{}) {
	renderer.name = name
}

func (renderer *renderedTemplate) RenderPartial(_ http.ResponseWriter, _, name string, _ interface{}) {
	renderer.name, renderer.partial = name, true
}

func TestHandlerRendersPageAndHTMXCreate(t *testing.T) {
	service := &handlerService{}
	renderer := &renderedTemplate{}
	handler := NewHandler(service, renderer, log.NewNoopLogger())
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	page := httptest.NewRecorder()
	router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, %q, nil))

	if page.Code != http.StatusOK || renderer.name != "page" || renderer.partial {
		t.Fatalf("page response = %%d, %%#v", page.Code, renderer)
	}

	request := httptest.NewRequest(http.MethodPost, %q, strings.NewReader(%q))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")

	created := httptest.NewRecorder()
	router.ServeHTTP(created, request)

	if created.Code != http.StatusOK || !service.created || renderer.name != "row" || !renderer.partial {
		t.Fatalf("create response = %%d, %%#v", created.Code, renderer)
	}
}%s
`, context.feature, validationImport, context.entity, context.entity, context.entity, context.entity, context.entity, context.entity, context.entity, context.route, context.route, validForm, validationTest)

	return formatGo(edit.Target, source)
}

func renderFormValues(fields []renderField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		value := "value"

		switch field.ParseKind {
		case "boolean":
			value = "true"
		case "date":
			value = "2026-09-28"
		case "integer":
			value = "1"
		case "timestamp":
			value = "2026-09-28T12:00"
		case "uuid":
			value = "550e8400-e29b-41d4-a716-446655440000"
		}

		values = append(values, field.Name+"="+value)
	}

	return strings.Join(values, "&")
}

func fieldsHaveKind(fields []renderField, kind string) bool {
	for _, field := range fields {
		if field.ParseKind == kind {
			return true
		}
	}

	return false
}

func fieldsNeedNormalization(fields []renderField) bool {
	for _, field := range fields {
		if field.ParseKind != "boolean" && field.ParseKind != "integer" && field.ParseKind != "date" && field.ParseKind != "timestamp" {
			return true
		}
	}

	return false
}
