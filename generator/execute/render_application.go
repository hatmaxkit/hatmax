// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
)

const (
	applicationModuleRecipe      = "application.module"
	applicationChecksumsRecipe   = "application.checksums"
	applicationIgnoreRecipe      = "application.ignore"
	applicationCommandsRecipe    = "application.commands"
	applicationMainRecipe        = "application.main"
	applicationConfigFileRecipe  = "application.config_file"
	applicationConfigRecipe      = "application.config"
	applicationLoggingRecipe     = "application.logging"
	applicationDatabaseRecipe    = "application.database"
	applicationCompositionRecipe = "application.composition"
	applicationTestsRecipe       = "application.composition_tests"
)

type applicationRenderContext struct {
	DisplayName   string
	ProjectSlug   string
	ModulePath    string
	HatmaxVersion string
	Environment   string
	Database      string
}

type applicationRecipe struct {
	target   string
	template string
	goSource bool
}

var applicationFoundationRecipes = map[string]applicationRecipe{
	applicationModuleRecipe: {
		target: "go.mod",
		template: `module {{.ModulePath}}

go 1.24.0

require (
	github.com/go-chi/chi/v5 v5.2.3
	hatmax.adrianpk.com {{.HatmaxVersion}}
)
`,
	},
	applicationChecksumsRecipe: {
		target: "go.sum",
		template: `github.com/go-chi/chi/v5 v5.2.3 h1:WQIt9uxdsAbgIYgid+BpYc+liqQZGMHRaUwp0JUcvdE=
github.com/go-chi/chi/v5 v5.2.3/go.mod h1:L2yAIGWB3H+phAw1NxKwWM+7eUH/lU8pOMm5hHcoops=
hatmax.adrianpk.com v0.5.0 h1:9LptUn0pjo9c1um7rhwwYYAhmDyg/ejjQBzm7X7UM3U=
hatmax.adrianpk.com v0.5.0/go.mod h1:X8IRNfizbKzpeyDUTs2iSuzpPvuF8zNa2ndkGC6c6Yg=
`,
	},
	applicationIgnoreRecipe: {
		target: ".gitignore",
		template: `/{{.ProjectSlug}}
/*.log
/*.pid
/coverage.out
/.env
/config.local.yaml
`,
	},
	applicationCommandsRecipe: {
		target: "Makefile",
		template: `BINARY := {{.ProjectSlug}}

.PHONY: build run test fmt lint clean

build:
	go build -o $(BINARY) .

run:
	go run .

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out
`,
	},
	applicationMainRecipe: {
		target:   "main.go",
		goSource: true,
		template: `package main

import (
	"context"
	"os"

	"{{.ModulePath}}/internal/application"
)

func main() {
	os.Exit(application.Run(context.Background(), os.Args))
}
`,
	},
	applicationConfigFileRecipe: {
		target: "config.yaml",
		template: `log:
  level: info

server:
  host: localhost
  port: ":8080"

database:
  host: localhost
  port: 5432
  user: dev
  password: dev
  database: {{.Database}}
  sslmode: disable
  schema: public
`,
	},
	applicationConfigRecipe: {
		target:   "internal/application/config.go",
		goSource: true,
		template: `package application

import (
	"fmt"

	"hatmax.adrianpk.com/config"
)

const environmentPrefix = "{{.Environment}}_"

func loadConfig(path string, args []string) (*config.Config, error) {
	cfg, err := config.Load(path, environmentPrefix, args)
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}
`,
	},
	applicationLoggingRecipe: {
		target:   "internal/application/logging.go",
		goSource: true,
		template: `package application

import (
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

func newLogger(cfg *config.Config) log.Logger {
	return log.NewLogger(cfg)
}
`,
	},
	applicationDatabaseRecipe: {
		target:   "internal/application/database.go",
		goSource: true,
		template: `package application

import (
	"embed"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
)

func newDatabase(assets embed.FS, cfg *config.Config, logger log.Logger) *db.Database {
	return db.New(assets, db.Postgres, cfg, logger)
}
`,
	},
	applicationCompositionRecipe: {
		target:   "internal/application/application.go",
		goSource: true,
		template: `package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

const (
	applicationName    = "{{.DisplayName}}"
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
		logger.Errorf("%s failed: %v", applicationName, err)

		return 1
	}

	return 0
}

func run(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	router, webComponents := newWeb(logger)
	database := newDatabase(assetsFS, cfg, logger)
	components := append([]any{database}, webComponents...)
	starts, stops, registrars := app.Setup(ctx, router, components...)

	if err := app.Start(ctx, logger, starts, stops, registrars, router); err != nil {
		return fmt.Errorf("start components: %w", err)
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

		return fmt.Errorf("serve HTTP: %w", err)
	}
}
`,
	},
	applicationTestsRecipe: {
		target:   "internal/application/application_test.go",
		goSource: true,
		template: `package application

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAppliesApplicationEnvironment(t *testing.T) {
	t.Setenv(environmentPrefix+"SERVER_PORT", ":9090")

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("database:\n  host: localhost\n  user: dev\n  database: {{.Database}}\nserver:\n  port: ':8080'\n")

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	cfg, err := loadConfig(path, []string{"{{.ProjectSlug}}"})
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Server.Port != ":9090" {
		t.Errorf("server port = %q, want :9090", cfg.Server.Port)
	}
}
`,
	},
}

// RenderApplicationFoundation renders the non-web portion of one canonical
// application scaffold. It returns only manifest-owned create mutations and
// never writes the target.
func RenderApplicationFoundation(value plan.Plan, manifest Manifest) ([]Mutation, error) {
	context, err := newApplicationRenderContext(value, manifest)
	if err != nil {
		return nil, err
	}

	mutations := make([]Mutation, 0, len(applicationFoundationRecipes))
	seen := make(map[string]struct{}, len(applicationFoundationRecipes))

	for _, edit := range manifest.Edits {
		recipe, exists := applicationFoundationRecipes[edit.Recipe]
		if !exists {
			continue
		}

		if recipe.target != edit.Target || edit.Kind != EditCreateFile {
			return nil, executionError("execution_recipe_target_invalid", edit.Target, "recipe %q requires create target %q", edit.Recipe, recipe.target)
		}

		if _, duplicate := seen[edit.Recipe]; duplicate {
			return nil, executionError("execution_recipe_duplicate", edit.Recipe, "application recipe occurs more than once")
		}

		var (
			content   []byte
			renderErr error
		)
		if edit.Recipe == applicationCompositionRecipe && len(value.Units) > 1 {
			content, renderErr = renderCompositeApplication(value)
		} else {
			content, renderErr = renderApplicationTemplate(context, recipe)
		}

		if renderErr != nil {
			return nil, executionError("execution_render_failed", edit.Target, "%v", renderErr)
		}

		mutations = append(mutations, Mutation{EditID: edit.ID, Kind: MutationCreate, Content: content})
		seen[edit.Recipe] = struct{}{}
	}

	for recipe := range applicationFoundationRecipes {
		if _, exists := seen[recipe]; !exists {
			return nil, executionError("execution_recipe_missing", recipe, "application manifest omits a foundation recipe")
		}
	}

	return mutations, nil
}

func newApplicationRenderContext(value plan.Plan, manifest Manifest) (applicationRenderContext, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return applicationRenderContext{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return applicationRenderContext{}, err
	}

	if value.SchemaVersion != plan.ApplicationSchemaVersion || value.Intent != intent.OperationCreateApplication || value.Application == nil {
		return applicationRenderContext{}, executionError("execution_intent_invalid", "intent", "application rendering requires a sealed create_application plan")
	}

	if manifest.Intent != value.Intent || manifest.PlanDigest != value.Digest || manifest.SourceFingerprint != value.SourceFingerprint {
		return applicationRenderContext{}, executionError("execution_identity_mismatch", "manifest", "manifest and application plan identities do not match")
	}

	version := value.HatmaxVersion
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	environment := strings.ToUpper(strings.ReplaceAll(value.Application.ProjectSlug, "-", "_"))
	database := strings.ReplaceAll(value.Application.ProjectSlug, "-", "_")

	return applicationRenderContext{
		DisplayName: value.Application.DisplayName, ProjectSlug: value.Application.ProjectSlug,
		ModulePath: value.Application.ModulePath, HatmaxVersion: version,
		Environment: environment, Database: database,
	}, nil
}

func renderApplicationTemplate(context applicationRenderContext, recipe applicationRecipe) ([]byte, error) {
	replacer := strings.NewReplacer(
		"{{.DisplayName}}", context.DisplayName,
		"{{.ProjectSlug}}", context.ProjectSlug,
		"{{.ModulePath}}", context.ModulePath,
		"{{.HatmaxVersion}}", context.HatmaxVersion,
		"{{.Environment}}", context.Environment,
		"{{.Database}}", context.Database,
	)
	rendered := replacer.Replace(recipe.template)

	if recipe.goSource {
		return formatGo(recipe.target, rendered)
	}

	return []byte(rendered), nil
}
