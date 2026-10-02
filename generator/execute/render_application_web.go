// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"
	"strings"

	"hatmax.adrianpk.com/generator/plan"
)

const (
	applicationWebRecipe      = "application.web"
	applicationLayoutRecipe   = "application.layout"
	applicationHomeRecipe     = "application.home"
	applicationStylesRecipe   = "application.styles"
	applicationWebTestsRecipe = "application.web_tests"
)

var applicationWebRecipes = map[string]applicationRecipe{
	applicationWebRecipe: {
		target:   "internal/application/web.go",
		goSource: true,
		template: `package application

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/middleware"
	"hatmax.adrianpk.com/ui"
	"hatmax.adrianpk.com/web"
)

//go:embed assets
var assetsFS embed.FS

type pageHandler struct {
	assets    embed.FS
	templates *web.TemplateManager
	logger    log.Logger
}

type homePage struct {
	ApplicationName string
}

func newWeb(logger log.Logger) (chi.Router, []any) {
	router := app.NewRouter(
		logger,
		app.WithMiddleware(middleware.DefaultStack()...),
		app.WithPing(),
	)
	templates := web.NewTemplateManager(assetsFS, logger, web.WithFuncMap(ui.FuncMap()))
	pages := &pageHandler{assets: assetsFS, templates: templates, logger: logger}

	return router, []any{templates, pages}
}

func (h *pageHandler) RegisterRoutes(router chi.Router) {
	staticFiles, err := fs.Sub(h.assets, "assets/static")
	if err != nil {
		h.logger.Errorf("cannot prepare static assets: %v", err)
	} else {
		router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	}

	router.Get("/", h.home)
}

func (h *pageHandler) home(response http.ResponseWriter, _ *http.Request) {
	h.templates.Render(response, "pages", "home", homePage{ApplicationName: applicationName})
}
`,
	},
	applicationLayoutRecipe: {
		target: "internal/application/assets/templates/layouts/base.html",
		template: `{{define "assets/templates/layouts/base.html"}}<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{template "title" .}}</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
  <main class="shell">
    {{template "content" .}}
  </main>
</body>
</html>{{end}}
`,
	},
	applicationHomeRecipe: {
		target: "internal/application/assets/templates/pages/home.html",
		template: `{{define "title"}}{{.ApplicationName}}{{end}}
{{define "content"}}
<section class="welcome">
  <p class="eyebrow">Hatmax application</p>
  <h1>{{.ApplicationName}}</h1>
  <p>The canonical server-rendered foundation is running.</p>
</section>
{{end}}
{{define "assets/templates/pages/home.html"}}{{template "assets/templates/layouts/base.html" .}}{{end}}
`,
	},
	applicationStylesRecipe: {
		target: "internal/application/assets/static/app.css",
		template: `:root {
  color-scheme: light dark;
  font-family: system-ui, sans-serif;
}

body {
  margin: 0;
  min-height: 100vh;
}

.shell {
  display: grid;
  min-height: 100vh;
  place-items: center;
  padding: 2rem;
}

.welcome {
  max-width: 42rem;
}

.eyebrow {
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
`,
	},
	applicationWebTestsRecipe: {
		target:   "internal/application/web_test.go",
		goSource: true,
		template: `package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/log"
)

func TestNeutralWebSurface(t *testing.T) {
	logger := log.NewTestLogger("error")
	router, components := newWeb(logger)
	starts, stops, registrars := app.Setup(context.Background(), router, components...)

	if err := app.Start(context.Background(), logger, starts, stops, registrars, router); err != nil {
		t.Fatalf("start web components: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if body := response.Body.String(); !strings.Contains(body, "{{.DisplayName}}") || !strings.Contains(body, "/static/app.css") {
		t.Errorf("landing page does not contain application identity and asset link:\n%s", body)
	}

	for index := len(stops) - 1; index >= 0; index-- {
		if err := stops[index](context.Background()); err != nil {
			t.Errorf("stop component: %v", err)
		}
	}
}
`,
	},
}

// RenderApplicationWeb renders the router, middleware, embedded templates,
// neutral landing page, static assets, and their behavioral test.
func RenderApplicationWeb(value plan.Plan, manifest Manifest) ([]Mutation, error) {
	context, err := newApplicationRenderContext(value, manifest)
	if err != nil {
		return nil, err
	}

	mutations := make([]Mutation, 0, len(applicationWebRecipes))
	seen := make(map[string]struct{}, len(applicationWebRecipes))

	for _, edit := range manifest.Edits {
		recipe, exists := applicationWebRecipes[edit.Recipe]
		if !exists {
			continue
		}

		if recipe.target != edit.Target || edit.Kind != EditCreateFile {
			return nil, executionError("execution_recipe_target_invalid", edit.Target, "recipe %q requires create target %q", edit.Recipe, recipe.target)
		}

		if _, duplicate := seen[edit.Recipe]; duplicate {
			return nil, executionError("execution_recipe_duplicate", edit.Recipe, "application recipe occurs more than once")
		}

		content, renderErr := renderApplicationTemplate(context, recipe)
		if renderErr != nil {
			return nil, executionError("execution_render_failed", edit.Target, "%v", renderErr)
		}

		mutations = append(mutations, Mutation{EditID: edit.ID, Kind: MutationCreate, Content: content})
		seen[edit.Recipe] = struct{}{}
	}

	for recipe := range applicationWebRecipes {
		if _, exists := seen[recipe]; !exists {
			return nil, executionError("execution_recipe_missing", recipe, "application manifest omits a web recipe")
		}
	}

	return mutations, nil
}

// RenderApplication renders every canonical scaffold recipe owned by the
// application archetype without performing filesystem mutation.
func RenderApplication(value plan.Plan, manifest Manifest) ([]Mutation, error) {
	foundation, err := RenderApplicationFoundation(value, manifest)
	if err != nil {
		return nil, err
	}

	web, err := RenderApplicationWeb(value, manifest)
	if err != nil {
		return nil, err
	}

	features, err := RenderApplicationFeatures(value, manifest)
	if err != nil {
		return nil, err
	}

	result := append(foundation, web...)

	result = append(result, features...)
	if len(result) != len(manifest.Edits) {
		return nil, executionError("execution_recipe_unsupported", "edits", "%d manifest edits have no application renderer", len(manifest.Edits)-len(result))
	}

	seen := make(map[string]struct{}, len(result))
	for _, mutation := range result {
		if _, exists := seen[mutation.EditID]; exists {
			return nil, executionError("execution_recipe_duplicate", mutation.EditID, "application edit was rendered more than once")
		}

		seen[mutation.EditID] = struct{}{}
	}

	return result, nil
}

func applicationRecipeForPath(target string) (string, applicationRecipe, bool) {
	for recipeID, recipe := range applicationFoundationRecipes {
		if recipe.target == target {
			return recipeID, recipe, true
		}
	}

	for recipeID, recipe := range applicationWebRecipes {
		if recipe.target == target {
			return recipeID, recipe, true
		}
	}

	return "", applicationRecipe{}, false
}

func validateApplicationRecipeCoverage(value plan.Plan) error {
	for _, effect := range value.AllowedEffects.Files {
		if _, _, exists := applicationRecipeForPath(effect.Path); exists {
			continue
		}

		if _, exists := applicationFeatureRecipe(value, effect.Path); !exists {
			return fmt.Errorf("planned path %q has no application recipe", effect.Path)
		}
	}

	wantMinimum := len(applicationFoundationRecipes) + len(applicationWebRecipes)
	if len(value.AllowedEffects.Files) < wantMinimum {
		return fmt.Errorf("application plan has %d file effects, scaffold recipes require at least %d", len(value.AllowedEffects.Files), wantMinimum)
	}

	return nil
}

func applicationRecipeID(target string) string {
	recipe, _, _ := applicationRecipeForPath(target)

	return strings.TrimSpace(recipe)
}
