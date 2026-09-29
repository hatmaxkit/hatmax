package execute

import (
	"fmt"
	"path"
	"strings"

	"hatmax.adrianpk.com/generator/plan"
)

const applicationSQLCRecipe = "application.sqlc"

func applicationFeatureRecipe(value plan.Plan, target string) (string, bool) {
	if target == "sqlc.yaml" {
		return applicationSQLCRecipe, true
	}

	unit, exists := applicationFeatureUnit(value, target)
	if !exists {
		return "", false
	}

	featureRoot := path.Join("internal/feat", unit.Feature)
	templateRoot := path.Join("internal/application/assets/templates", unit.Feature)
	recipes := map[string]string{
		path.Join(featureRoot, "model.go"):               "server_rendered_crud.model",
		path.Join(featureRoot, "store.go"):               "server_rendered_crud.store_contract",
		path.Join(featureRoot, "postgres_store.go"):      "server_rendered_crud.postgres_store",
		path.Join(featureRoot, "service.go"):             "server_rendered_crud.service",
		path.Join(featureRoot, "handler.go"):             "server_rendered_crud.handler",
		path.Join(featureRoot, "model_test.go"):          "server_rendered_crud.model_tests",
		path.Join(featureRoot, "service_test.go"):        "server_rendered_crud.service_tests",
		path.Join(featureRoot, "handler_test.go"):        "server_rendered_crud.handler_tests",
		path.Join(featureRoot, "postgres_store_test.go"): "server_rendered_crud.postgres_store_tests",
		path.Join("db/queries", unit.Feature+".sql"):     "server_rendered_crud.queries",
		path.Join(templateRoot, "page.html"):             "server_rendered_crud.page_template",
		path.Join(templateRoot, "form.html"):             "server_rendered_crud.form_template",
		path.Join(templateRoot, "row.html"):              "server_rendered_crud.row_template",
	}

	if recipe := recipes[target]; recipe != "" {
		return recipe, true
	}

	if strings.HasPrefix(target, "assets/migration/postgres/") && strings.HasSuffix(target, "-"+strings.ReplaceAll(unit.Feature, "_", "-")+".sql") {
		return "server_rendered_crud.migration", true
	}

	return "", false
}

func applicationFeatureUnit(value plan.Plan, target string) (plan.Unit, bool) {
	for _, unit := range value.Units {
		if unit.Feature == "" {
			continue
		}

		for _, effect := range unit.Effects {
			if effect.Path == target {
				return unit, true
			}
		}
	}

	return plan.Unit{}, false
}

// RenderApplicationFeatures renders all initial feature units against the
// final composite manifest. Shared application wiring is rendered once from
// the complete unit set.
func RenderApplicationFeatures(value plan.Plan, manifest Manifest) ([]Mutation, error) {
	if len(value.Units) <= 1 {
		return []Mutation{}, nil
	}

	renderers := make(map[string]recipeRenderer, len(domainRenderers)+len(transportRenderers))
	for recipe, renderer := range domainRenderers {
		renderers[recipe] = renderer
	}
	for recipe, renderer := range transportRenderers {
		renderers[recipe] = renderer
	}

	result := make([]Mutation, 0)
	for _, edit := range manifest.Edits {
		if edit.Recipe == applicationSQLCRecipe {
			result = append(result, Mutation{EditID: edit.ID, Kind: MutationCreate, Content: []byte(applicationSQLCConfiguration)})

			continue
		}

		renderer, exists := renderers[edit.Recipe]
		if !exists {
			continue
		}

		unit, exists := applicationFeatureUnit(value, edit.Target)
		if !exists {
			return nil, executionError("execution_recipe_target_invalid", edit.Target, "initial-feature recipe has no owning unit")
		}

		context, err := newApplicationFeatureRenderContext(value, manifest, unit)
		if err != nil {
			return nil, err
		}

		content, err := renderer(context, edit)
		if err != nil {
			return nil, executionError("execution_render_failed", edit.Target, "%v", err)
		}

		result = append(result, Mutation{EditID: edit.ID, Kind: MutationCreate, Content: content})
	}

	return result, nil
}

func newApplicationFeatureRenderContext(value plan.Plan, manifest Manifest, unit plan.Unit) (renderContext, error) {
	fields := make([]renderField, 0, len(unit.Domain.Fields))
	for _, field := range unit.Domain.Fields {
		rendered, err := newRenderField(field)
		if err != nil {
			return renderContext{}, err
		}

		fields = append(fields, rendered)
	}

	label := unit.Domain.Label
	if strings.TrimSpace(label) == "" {
		label = unit.Domain.Entity + "s"
	}

	table := strings.NewReplacer("/", "_", "-", "_").Replace(strings.TrimPrefix(unit.Domain.Route, "/"))

	return renderContext{
		plan: value, manifest: manifest, modulePath: value.Application.ModulePath,
		entity: unit.Domain.Entity, feature: unit.Feature, table: table,
		route: unit.Domain.Route, label: label, plural: exportedName(table), fields: fields,
	}, nil
}

func renderCompositeApplication(value plan.Plan) ([]byte, error) {
	var imports strings.Builder
	var construction strings.Builder
	var dependencies strings.Builder

	for _, unit := range value.Units {
		if unit.Feature == "" {
			continue
		}

		prefix := applicationFeaturePrefix(unit.Feature)
		alias := prefix + "feat"
		fmt.Fprintf(&imports, "\t%s %q\n", alias, value.Application.ModulePath+"/internal/feat/"+unit.Feature)
		fmt.Fprintf(&construction, "\t%sStore := %s.NewPostgresStore(database)\n", prefix, alias)
		fmt.Fprintf(&construction, "\t%sService := %s.NewService(%sStore)\n", prefix, alias, prefix)
		fmt.Fprintf(&construction, "\t%sHandler := %s.NewHandler(%sService, templates, logger)\n", prefix, alias, prefix)
		fmt.Fprintf(&dependencies, ", %sStore, %sHandler", prefix, prefix)
	}

	source := fmt.Sprintf(`package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

%s
	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/web"
)

const (
	applicationName    = %q
	applicationVersion = "0.1.0"
)

func Run(ctx context.Context, args []string) int {
	cfg, err := loadConfig("config.yaml", args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	logger := newLogger(cfg)
	if err := run(ctx, cfg, logger); err != nil {
		logger.Errorf("%%s failed: %%v", applicationName, err)

		return 1
	}

	return 0
}

func run(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	router, webComponents := newWeb(logger)
	templates := web.NewTemplateManager(assetsFS, logger)
	database := newDatabase(assetsFS, cfg, logger)
	migrator := db.NewMigrator(database, assetsFS, db.Postgres, logger)
%s
	components := append([]any{database, migrator, templates%s}, webComponents...)
	starts, stops, registrars := app.Setup(ctx, router, components...)

	if err := app.Start(ctx, logger, starts, stops, registrars, router); err != nil {
		return fmt.Errorf("start components: %%w", err)
	}

	server := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErrors := make(chan error, 1)

	go func() {
		serveErrors <- server.ListenAndServe()
	}()

	stopContext, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	select {
	case <-stopContext.Done():
		app.Shutdown(server, logger, stops)

		return nil
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("serve HTTP: %%w", err)
	}
}
`, imports.String(), value.Application.DisplayName, construction.String(), dependencies.String())

	return formatGo("internal/application/application.go", source)
}

func applicationFeaturePrefix(feature string) string {
	parts := strings.Split(feature, "_")
	if len(parts) == 0 {
		return feature
	}

	var result strings.Builder
	result.WriteString(parts[0])
	for _, part := range parts[1:] {
		result.WriteString(exportedName(part))
	}

	return result.String()
}

const applicationSQLCConfiguration = `version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries"
    schema: "assets/migration/postgres"
    gen:
      go:
        package: "dal"
        out: "internal/dal"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
`
