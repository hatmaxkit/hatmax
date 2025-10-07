package hatmax

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"

	_ "github.com/adrianpk/hatmax/pkg/lib/hm"
)

// FieldTemplateData holds data for a single field in the template.
type FieldTemplateData struct {
	Name        string
	Type        string
	JSONTag     string
	IsID        bool
	Validations []FieldValidationData
}

// FieldValidationData holds data for a single validation rule.
type FieldValidationData struct {
	Name  string
	Value string
}

// ModelTemplateData holds all data needed to render a model template.
type ModelTemplateData struct {
	PackageName  string
	ModelName    string
	Audit        bool
	Fields       []FieldTemplateData
	NeedsFmt     bool
	NeedsStrconv bool
}

// HandlerTemplateData holds all data needed to render a handler template.
type HandlerTemplateData struct {
	PackageName       string
	ModelName         string
	ModelPlural       string
	ModelLower        string
	ModelPluralLower  string
	AuthEnabled       bool
	Audit             bool
	ModulePath        string
	IsChildCollection bool
}

type ModelGenerator struct {
	Config                   Config
	OutputDir                string
	DevMode                  bool
	Template                 *template.Template
	RepoInterfaceTemplate    *template.Template
	ServiceInterfaceTemplate *template.Template
	SQLiteRepoTemplate       *template.Template
	SQLiteQueriesTemplate    *template.Template
	MongoRepoTemplate        *template.Template
	HandlerTemplate          *template.Template
	ValidatorTemplate        *template.Template
	MainTemplate             *template.Template
	ConfigTemplate           *template.Template
	ConfigYAMLTemplate       *template.Template
	XParamsTemplate          *template.Template
	MakefileTemplate         *template.Template
	AggregateRootTemplate    *template.Template
	ChildCollectionTemplate  *template.Template
}

// NewModelGenerator creates a new ModelGenerator.
func NewModelGenerator(config Config, outputDir string, devMode bool, assetsFS fs.FS) (*ModelGenerator, error) {
	tmplFS, err := fs.Sub(assetsFS, "assets/templates")
	if err != nil {
		log.Fatalf("cannot create sub-filesystem for templates: %v", err)
	}

	tmpl, err := template.New("model.tmpl").ParseFS(tmplFS, "model.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse model template: %w", err)
	}

	repoInterfaceTmpl, err := template.New("repo_interface.tmpl").ParseFS(tmplFS, "repo_interface.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse repository interface template: %w", err)
	}

	serviceInterfaceTmpl, err := template.New("service_interface.tmpl").ParseFS(tmplFS, "service_interface.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse service interface template: %w", err)
	}

	sqliteRepoTmpl, err := template.New("repo_sqlite.tmpl").ParseFS(tmplFS, "repo_sqlite.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse SQLite repository template: %w", err)
	}

	sqliteQueriesTmpl, err := template.New("queries_sqlite.tmpl").ParseFS(tmplFS, "queries_sqlite.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse SQLite queries template: %w", err)
	}

	mongoRepoTmpl, err := template.New("repo_mongo.tmpl").ParseFS(tmplFS, "repo_mongo.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse MongoDB repository template: %w", err)
	}

	handlerTmpl, err := template.New("handler.tmpl").ParseFS(tmplFS, "handler.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse handler template: %w", err)
	}

	validatorTmpl, err := template.New("validator.tmpl").ParseFS(tmplFS, "validator.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse validator template: %w", err)
	}

	mainTmpl, err := template.New("main.tmpl").ParseFS(tmplFS, "main.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse main template: %w", err)
	}

	configTmpl, err := template.New("config.go.tmpl").ParseFS(tmplFS, "config.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse config go template: %w", err)
	}

	configYAMLTmpl, err := template.New("config.yaml.tmpl").ParseFS(tmplFS, "config.yaml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse config yaml template: %w", err)
	}

	xparamsTmpl, err := template.New("xparams.go.tmpl").ParseFS(tmplFS, "xparams.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse xparams template: %w", err)
	}

	makefileTmpl, err := template.New("Makefile.tmpl").ParseFS(tmplFS, "Makefile.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse Makefile template: %w", err)
	}

	aggregateRootTmpl, err := template.New("aggregate_root.tmpl").ParseFS(tmplFS, "aggregate_root.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse aggregate root template: %w", err)
	}

	childCollectionTmpl, err := template.New("child_collection.tmpl").ParseFS(tmplFS, "child_collection.tmpl")
	if err != nil {
		return nil, fmt.Errorf("cannot parse child collection template: %w", err)
	}

	return &ModelGenerator{
			Config:                   config,
			OutputDir:                outputDir,
			DevMode:                  devMode,
			Template:                 tmpl,
			RepoInterfaceTemplate:    repoInterfaceTmpl,
			ServiceInterfaceTemplate: serviceInterfaceTmpl,
			SQLiteRepoTemplate:       sqliteRepoTmpl,
			SQLiteQueriesTemplate:    sqliteQueriesTmpl,
			MongoRepoTemplate:        mongoRepoTmpl,
			HandlerTemplate:          handlerTmpl,
			ValidatorTemplate:        validatorTmpl,
			MainTemplate:             mainTmpl,
			ConfigTemplate:           configTmpl,
			ConfigYAMLTemplate:       configYAMLTmpl,
			XParamsTemplate:          xparamsTmpl,
			MakefileTemplate:         makefileTmpl,
			AggregateRootTemplate:    aggregateRootTmpl,
			ChildCollectionTemplate:  childCollectionTmpl,
		},
		nil
}

// GenerateModels generates the Go model files based on the configuration.
func (mg *ModelGenerator) GenerateModels() error {
	for serviceName, service := range mg.Config.Services {
		for modelName, model := range service.Models {
			if model.Fields == nil {
				model.Fields = make(map[string]Field)
			}
			fmt.Printf("  - Generating model: %s/%s\n", serviceName, modelName)

			packageName := serviceName
			modelFileName := strings.ToLower(modelName) + ".go"
			modelPath := filepath.Join(mg.OutputDir, "internal", serviceName, modelFileName)

			data := ModelTemplateData{
				PackageName: packageName,
				ModelName:   modelName,
				Audit:       false,
				Fields:      []FieldTemplateData{},
			}
			if model.Options != nil {
				data.Audit = model.Options.Audit
			}

			var needsFmt bool
			var needsStrconv bool

			for fieldName, field := range model.Fields {
				goType := mapGoType(field.Type)
				fieldData := FieldTemplateData{
					Name:        strings.Title(fieldName),
					Type:        goType,
					JSONTag:     toSnakeCase(fieldName),
					IsID:        false,
					Validations: []FieldValidationData{},
				}

				for _, v := range field.Validations {
					fieldData.Validations = append(fieldData.Validations, FieldValidationData{
						Name:  v.Name,
						Value: v.Value,
					})
					switch v.Name {
					case "min_length", "max_length", "min", "max":
						needsFmt = true
						needsStrconv = true
					}
				}
				data.Fields = append(data.Fields, fieldData)
			}

			data.NeedsFmt = needsFmt
			data.NeedsStrconv = needsStrconv

			dir := filepath.Dir(modelPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for model %s: %w", modelName, err)
			}

			file, err := os.Create(modelPath)
			if err != nil {
				return fmt.Errorf("cannot create model file %s: %w", modelPath, err)
			}
			defer file.Close()

			if err := mg.Template.Execute(file, data); err != nil {
				return fmt.Errorf("cannot execute model template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", modelPath)
		}
	}
	return nil
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// GenerateRepoInterfaces generates the Go repository interface files based on the configuration.
func (mg *ModelGenerator) GenerateRepoInterfaces() error {
	for serviceName, service := range mg.Config.Services {
		for modelName := range service.Models {
			fmt.Printf("  - Generating repository interface: %s/%sRepo\n", serviceName, modelName)

			packageName := serviceName // For now, package name is the service name
			repoFileName := strings.ToLower(modelName) + "repo.go"
			repoPath := filepath.Join(mg.OutputDir, "internal", serviceName, repoFileName)

			data := struct {
				PackageName string
				ModelName   string
			}{
				PackageName: packageName,
				ModelName:   modelName,
			}

			dir := filepath.Dir(repoPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for repository interface %s: %w", modelName, err)
			}

			file, err := os.Create(repoPath)
			if err != nil {
				return fmt.Errorf("cannot create repository interface file %s: %w", repoPath, err)
			}
			defer file.Close()

			if err := mg.RepoInterfaceTemplate.Execute(file, data); err != nil {
				return fmt.Errorf("cannot execute repository interface template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", repoPath)
		}
	}
	return nil
}

// GenerateServiceInterfaces generates the Go service interface files based on the configuration.
func (mg *ModelGenerator) GenerateServiceInterfaces() error {
	for serviceName, service := range mg.Config.Services {
		for modelName := range service.Models {
			fmt.Printf("  - Generating service interface: %s/%sService\n", serviceName, modelName)

			packageName := serviceName // For now, package name is the service name
			serviceFileName := strings.ToLower(modelName) + "service.go"
			servicePath := filepath.Join(mg.OutputDir, "internal", serviceName, serviceFileName)

			data := struct {
				PackageName string
				ModelName   string
			}{
				PackageName: packageName,
				ModelName:   modelName,
			}

			dir := filepath.Dir(servicePath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for service interface %s: %w", modelName, err)
			}

			file, err := os.Create(servicePath)
			if err != nil {
				return fmt.Errorf("cannot create service interface file %s: %w", servicePath, err)
			}
			defer file.Close()

			if err := mg.ServiceInterfaceTemplate.Execute(file, data); err != nil {
				return fmt.Errorf("cannot execute service interface template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", servicePath)
		}
	}
	return nil
}

// GenerateSQLiteRepoImplementations generates the SQLite repository implementation files based on the configuration.
func (mg *ModelGenerator) GenerateSQLiteRepoImplementations() error {
	for serviceName, service := range mg.Config.Services {
		// Only generate for services that use sqlite
		if !contains(service.RepoImpl, "sqlite") {
			continue
		}

		for modelName, model := range service.Models {
			fmt.Printf("  - Generating SQLite repository implementation: %s/%sRepo (sqlite)\n", serviceName, modelName)

			packageName := "sqlite"
			repoFileName := strings.ToLower(modelName) + "repo.go"
			queriesFileName := strings.ToLower(modelName) + "_queries.go"

			repoPath := filepath.Join(mg.OutputDir, "internal", "sqlite", repoFileName)
			queriesPath := filepath.Join(mg.OutputDir, "internal", "sqlite", queriesFileName)

			data := struct {
				PackageName       string
				ModelName         string
				TableName         string
				FieldNames        string
				FieldPlaceholders string
				FieldAssignments  string
				FieldValues       string
				FieldPointers     string
				ModulePath        string
				ServiceName       string
				ModelLower        string
			}{
				PackageName: packageName,
				ModelName:   modelName,
				TableName:   strings.ToLower(modelName) + "s", // TODO: Use a pluralize lib
				ModulePath:  mg.Config.ModulePath,
				ServiceName: serviceName,
				ModelLower:  strings.ToLower(modelName),
			}

			var fieldNames []string
			var fieldPlaceholders []string
			var fieldAssignments []string
			var fieldValues []string
			var fieldPointers []string

			for fieldName, field := range model.Fields {
				_ = field
				fieldNames = append(fieldNames, fieldName)
				fieldPlaceholders = append(fieldPlaceholders, "?")
				fieldAssignments = append(fieldAssignments, fmt.Sprintf("%s = ?", fieldName))
				fieldValues = append(fieldValues, fmt.Sprintf("item.%s", strings.Title(fieldName)))
				fieldPointers = append(fieldPointers, fmt.Sprintf("&item.%s", strings.Title(fieldName)))
			}

			data.FieldNames = strings.Join(fieldNames, ", ")
			data.FieldPlaceholders = strings.Join(fieldPlaceholders, ", ")
			data.FieldAssignments = strings.Join(fieldAssignments, ", ")
			data.FieldValues = strings.Join(fieldValues, ", ")
			data.FieldPointers = strings.Join(fieldPointers, ", ")

			dir := filepath.Dir(repoPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for SQLite repository %s: %w", modelName, err)
			}

			queriesFile, err := os.Create(queriesPath)
			if err != nil {
				return fmt.Errorf("cannot create SQLite queries file %s: %w", queriesPath, err)
			}
			defer queriesFile.Close()

			if err := mg.SQLiteQueriesTemplate.Execute(queriesFile, data); err != nil {
				return fmt.Errorf("cannot execute SQLite queries template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", queriesPath)

			repoFile, err := os.Create(repoPath)
			if err != nil {
				return fmt.Errorf("cannot create SQLite repository file %s: %w", repoPath, err)
			}
			defer repoFile.Close()

			if err := mg.SQLiteRepoTemplate.Execute(repoFile, data); err != nil {
				return fmt.Errorf("cannot execute SQLite repository template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", repoPath)
		}
	}
	return nil
}

// GenerateMongoRepoImplementations generates the MongoDB repository implementation files based on the configuration.
func (mg *ModelGenerator) GenerateMongoRepoImplementations() error {
	for serviceName, service := range mg.Config.Services {
		if !contains(service.RepoImpl, "mongo") {
			continue
		}

		for modelName := range service.Models {
			fmt.Printf("  - Generating MongoDB repository implementation: %s/%sRepo (mongo)\n", serviceName, modelName)

			packageName := "mongo"
			repoFileName := strings.ToLower(modelName) + "repo.go"

			repoPath := filepath.Join(mg.OutputDir, "internal", "mongo", repoFileName)

			data := struct {
				PackageName string
				ModelName   string
				TableName   string
			}{
				PackageName: packageName,
				ModelName:   modelName,
				TableName:   strings.ToLower(modelName) + "s",
			}

			dir := filepath.Dir(repoPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for MongoDB repository %s: %w", modelName, err)
			}

			repoFile, err := os.Create(repoPath)
			if err != nil {
				return fmt.Errorf("cannot create MongoDB repository file %s: %w", repoPath, err)
			}
			defer repoFile.Close()

			if err := mg.MongoRepoTemplate.Execute(repoFile, data); err != nil {
				return fmt.Errorf("cannot execute MongoDB repository template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", repoPath)
		}
	}
	return nil
}

// GenerateHandlers generates the Go handler files based on the configuration.
func (mg *ModelGenerator) GenerateHandlers() error {
	for serviceName, service := range mg.Config.Services {
		for modelName, model := range service.Models {
			fmt.Printf("  - Generating handler: %s/%sHandler\n", serviceName, modelName)

			packageName := serviceName
			handlerFileName := strings.ToLower(modelName) + "handler.go"
			handlerPath := filepath.Join(mg.OutputDir, "internal", serviceName, handlerFileName)

			modelPlural := pluralize(modelName)

			isChildCollection := false
			for _, aggregate := range service.Aggregates {
				for _, child := range aggregate.Children {
					if child.Of == modelName {
						isChildCollection = true
						break
					}
				}
				if isChildCollection {
					break
				}
			}

			data := HandlerTemplateData{
				PackageName:       packageName,
				ModelName:         modelName,
				ModelPlural:       modelPlural,
				ModelLower:        strings.ToLower(modelName),
				ModelPluralLower:  strings.ToLower(modelPlural),
				AuthEnabled:       service.Auth != nil && service.Auth.Enabled,
				Audit:             false,
				ModulePath:        mg.Config.ModulePath,
				IsChildCollection: isChildCollection,
			}
			if model.Options != nil {
				data.Audit = model.Options.Audit
			}

			dir := filepath.Dir(handlerPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for handler %s: %w", modelName, err)
			}

			file, err := os.Create(handlerPath)
			if err != nil {
				return fmt.Errorf("cannot create handler file %s: %w", handlerPath, err)
			}
			defer file.Close()

			if err := mg.HandlerTemplate.Execute(file, data); err != nil {
				return fmt.Errorf("cannot execute handler template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", handlerPath)
		}
	}
	return nil
}

// GenerateValidators generates the Go validator files based on the configuration.
func (mg *ModelGenerator) GenerateValidators() error {
	for serviceName, service := range mg.Config.Services {
		for modelName, model := range service.Models {
			fmt.Printf("  - Generating validator: %s/%sValidator\n", serviceName, modelName)

			packageName := serviceName // For now, package name is the service name
			validatorFileName := strings.ToLower(modelName) + "validator.go"
			validatorPath := filepath.Join(mg.OutputDir, "internal", serviceName, validatorFileName)

			data := ModelTemplateData{
				PackageName: packageName,
				ModelName:   modelName,
				Audit:       false,
				Fields:      []FieldTemplateData{},
			}
			if model.Options != nil {
				data.Audit = model.Options.Audit
			}

			var needsFmt bool
			var needsStrconv bool

			for fieldName, field := range model.Fields {
				goType := mapGoType(field.Type)
				fieldData := FieldTemplateData{
					Name:        strings.Title(fieldName),
					Type:        goType,
					JSONTag:     toSnakeCase(fieldName),
					IsID:        false,
					Validations: []FieldValidationData{},
				}

				for _, v := range field.Validations {
					fieldData.Validations = append(fieldData.Validations, FieldValidationData{
						Name:  v.Name,
						Value: v.Value,
					})
					switch v.Name {
					case "min_length", "max_length", "min", "max":
						needsFmt = true
						needsStrconv = true
					}
				}
				data.Fields = append(data.Fields, fieldData)
			}

			data.NeedsFmt = needsFmt
			data.NeedsStrconv = needsStrconv

			dir := filepath.Dir(validatorPath)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory for validator %s: %w", modelName, err)
			}

			file, err := os.Create(validatorPath)
			if err != nil {
				return fmt.Errorf("cannot create validator file %s: %w", validatorPath, err)
			}
			defer file.Close()

			if err := mg.ValidatorTemplate.Execute(file, data); err != nil {
				return fmt.Errorf("cannot execute validator template for %s: %w", modelName, err)
			}
			fmt.Printf("    - Created %s\n", validatorPath)
		}
	}
	return nil
}

// GenerateMain generates the main.go file for the generated application.
func (mg *ModelGenerator) GenerateMain() error {
	if mg.MainTemplate == nil {
		return fmt.Errorf("main template not initialized")
	}

	mainPath := filepath.Join(mg.OutputDir, "main.go")

	serviceNames := make([]string, 0, len(mg.Config.Services))
	for name := range mg.Config.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	type mainTemplateService struct {
		Name   string
		Models []string
	}

	services := make([]mainTemplateService, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		service := mg.Config.Services[serviceName]
		modelNames := make([]string, 0, len(service.Models))
		for modelName := range service.Models {
			modelNames = append(modelNames, modelName)
		}
		sort.Strings(modelNames)

		services = append(services, mainTemplateService{
			Name:   serviceName,
			Models: modelNames,
		})
	}

	data := struct {
		ModulePath string
		Services   []mainTemplateService
	}{
		ModulePath: mg.Config.ModulePath,
		Services:   services,
	}

	if err := mg.generateFile(mg.MainTemplate, mainPath, data); err != nil {
		return fmt.Errorf("cannot generate main.go: %w", err)
	}

	fmt.Printf("  - Created %s\n", mainPath)
	return nil
}

// GenerateConfigAndXParams generates the configuration files and XParams struct for the application.
func (mg *ModelGenerator) GenerateConfigAndXParams() error {
	fmt.Println("  - Generating configuration files and XParams...")

	configGoPath := filepath.Join(mg.OutputDir, "internal", "config", "config.go")
	data := struct {
		ModulePath string
	}{
		ModulePath: mg.Config.ModulePath,
	}
	if err := mg.generateFile(mg.ConfigTemplate, configGoPath, data); err != nil {
		return fmt.Errorf("cannot generate config.go: %w", err)
	}
	fmt.Printf("    - Created %s\n", configGoPath)

	xparamsGoPath := filepath.Join(mg.OutputDir, "internal", "config", "xparams.go")
	if err := mg.generateFile(mg.XParamsTemplate, xparamsGoPath, data); err != nil {
		return fmt.Errorf("cannot generate xparams.go: %w", err)
	}
	fmt.Printf("    - Created %s\n", xparamsGoPath)

	configYAMLPath := filepath.Join(mg.OutputDir, "config.yaml")
	if err := mg.generateFile(mg.ConfigYAMLTemplate, configYAMLPath, nil); err != nil {
		return fmt.Errorf("cannot generate config.yaml: %w", err)
	}
	fmt.Printf("    - Created %s\n", configYAMLPath)

	return nil
}

// generateFile is a helper to execute a template and write to a file.
func (mg *ModelGenerator) generateFile(tmpl *template.Template, path string, data any) error {
	if tmpl == nil {
		return fmt.Errorf("template for %s not initialized", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create file %s: %w", path, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("cannot execute template for %s: %w", path, err)
	}
	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func sanitizeIdentifier(name string) string {
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('_')
		}
	}

	identifier := builder.String()
	if identifier == "" {
		return "svc"
	}

	if !unicode.IsLetter(rune(identifier[0])) {
		return "svc_" + identifier
	}

	return identifier
}

// GenerateAggregateModels generates the Go structs for aggregate roots and their child collections.
func (mg *ModelGenerator) GenerateAggregateModels() error {
	for serviceName, service := range mg.Config.Services {
		for aggregateName, aggregate := range service.Aggregates {
			fmt.Printf("  - Generating aggregate root: %s/%s\n", serviceName, aggregateName)

			packageName := serviceName
			aggregateFileName := strings.ToLower(aggregateName) + ".go"
			aggregatePath := filepath.Join(mg.OutputDir, "internal", serviceName, aggregateFileName)

			data := struct {
				PackageName   string
				AggregateName string
				VersionField  string
				Fields        []FieldTemplateData
				Audit         bool
				Children      []ChildTemplateData
			}{
				PackageName:   packageName,
				AggregateName: aggregateName,
				VersionField:  aggregate.VersionField,
				Audit:         aggregate.Audit,
				Children:      []ChildTemplateData{},
			}

			for fieldName, field := range aggregate.Fields {
				data.Fields = append(data.Fields, FieldTemplateData{
					Name:    capitalizeFirst(fieldName),
					Type:    mapGoType(field.Type),
					JSONTag: toSnakeCase(fieldName),
				})
			}

			// Populate children
			for childName, child := range aggregate.Children {
				childModel := service.Models[child.Of] // Get the actual model definition
				if childModel.Fields == nil {
					return fmt.Errorf("child model %s for aggregate %s has no fields defined", child.Of, aggregateName)
				}

				childData := ChildTemplateData{
					ChildModelName: child.Of,
					Name:           capitalizeFirst(childName), // Name of the slice field in the root
					JSONTag:        toSnakeCase(childName),
					Audit:          child.Audit,
					Fields:         []FieldTemplateData{},
				}

				for fieldName, field := range childModel.Fields {
					childData.Fields = append(childData.Fields, FieldTemplateData{
						Name:    capitalizeFirst(fieldName),
						Type:    mapGoType(field.Type),
						JSONTag: toSnakeCase(fieldName),
					})
				}
				data.Children = append(data.Children, childData)
			}

			if err := mg.generateFile(mg.AggregateRootTemplate, aggregatePath, data); err != nil {
				return fmt.Errorf("cannot execute aggregate root template for %s: %w", aggregateName, err)
			}
			fmt.Printf("    - Created %s\n", aggregatePath)

			// Generate child collection structs (if they are not top-level models)
			for _, child := range aggregate.Children {
				// Check if this child model is already generated as a top-level model
				// For now, we assume child models are defined under service.Models
				// and will be generated as part of the regular model generation.
				// If a child model is only used as a child, it might need its own file.
				// For simplicity, we'll generate it as a separate file for now.
				fmt.Printf("  - Generating child collection struct: %s/%s\n", serviceName, child.Of)

				childFileName := strings.ToLower(child.Of) + ".go"
				childPath := filepath.Join(mg.OutputDir, "internal", serviceName, childFileName)

				childModel := service.Models[child.Of]
				if childModel.Fields == nil {
					return fmt.Errorf("child model %s for aggregate %s has no fields defined", child.Of, aggregateName)
				}

				childStructData := struct {
					PackageName    string
					ChildModelName string
					Fields         []FieldTemplateData
					Audit          bool
				}{
					PackageName:    packageName,
					ChildModelName: child.Of,
					Audit:          child.Audit,
					Fields:         []FieldTemplateData{},
				}

				for fieldName, field := range childModel.Fields {
					childStructData.Fields = append(childStructData.Fields, FieldTemplateData{
						Name:    capitalizeFirst(fieldName),
						Type:    mapGoType(field.Type),
						JSONTag: toSnakeCase(fieldName),
					})
				}

				if err := mg.generateFile(mg.ChildCollectionTemplate, childPath, childStructData); err != nil {
					return fmt.Errorf("cannot execute child collection template for %s: %w", child.Of, err)
				}
				fmt.Printf("    - Created %s\n", childPath)
			}
		}
	}
	return nil
}

// ChildTemplateData holds data for a child collection within an aggregate root.
type ChildTemplateData struct {
	ChildModelName string
	Name           string // Name of the slice field in the root
	JSONTag        string
	Audit          bool
	Fields         []FieldTemplateData
}

// GenerateGoMod generates the go.mod file for the generated application.
func (mg *ModelGenerator) GenerateGoMod() error {
	goModPath := filepath.Join(mg.OutputDir, "go.mod")

	// Generate go.mod with module name and Go version
	goModContent := fmt.Sprintf("module %s\n\ngo 1.23\n", mg.Config.ModulePath)

	// In dev mode, add the hm library dependency and replace directive
	if mg.DevMode {
		goModContent += "\nrequire (\n"
		goModContent += "\tgithub.com/adrianpk/hatmax v0.0.0-00010101000000-000000000000\n"
		goModContent += ")\n"
		goModContent += "\n// Development mode: use local hatmax library\n"
		// In dev mode, services are always generated at examples/ref/services/[service]/
		// So the path from service to hatmax root is always ../../../..
		goModContent += "replace github.com/adrianpk/hatmax => ../../../..\n"
	} else {
		// In production mode, let `go mod tidy` handle dependency management
		goModContent += "\n// Run 'go mod tidy' to add dependencies automatically\n"
	}

	err := os.WriteFile(goModPath, []byte(goModContent), 0o644)
	if err != nil {
		return fmt.Errorf("cannot write go.mod for generated app: %w", err)
	}
	fmt.Printf("  - Created %s\n", goModPath)
	return nil
}

// PostGenerationCleanup runs post-generation tasks like go mod tidy, gofmt, and goimports
func (mg *ModelGenerator) PostGenerationCleanup() error {
	// Change to the output directory for running go commands
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %w", err)
	}

	if err := os.Chdir(mg.OutputDir); err != nil {
		return fmt.Errorf("cannot change to output directory %s: %w", mg.OutputDir, err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	// Run go mod tidy
	fmt.Println("  - Running go mod tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	// Run gofmt on all .go files
	fmt.Println("  - Running gofmt...")
	cmd = exec.Command("gofmt", "-w", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: gofmt failed: %v\n", err)
		// Don't return error for gofmt failure, it's not critical
	}

	// Run goimports if available
	fmt.Println("  - Running goimports...")
	cmd = exec.Command("goimports", "-w", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: goimports failed (may not be installed): %v\n", err)
		// Don't return error for goimports failure, it's optional
	}

	return nil
}

// GenerateMakefile generates the Makefile for the generated application.
func (mg *ModelGenerator) GenerateMakefile(serviceName string) error {
	fmt.Printf("  - Generating Makefile for service %s...\n", serviceName)

	makefilePath := filepath.Join(mg.OutputDir, "Makefile")

	data := struct {
		ServiceName string
	}{
		ServiceName: serviceName,
	}

	if err := mg.generateFile(mg.MakefileTemplate, makefilePath, data); err != nil {
		return fmt.Errorf("cannot generate Makefile for service %s: %w", serviceName, err)
	}
	fmt.Printf("    - Created %s\n", makefilePath)
	return nil
}

// mapGoType maps YAML types to Go types.
// TODO: Use a pluralizer lib
func mapGoType(yamlType string) string {
	switch yamlType {
	case "text", "string", "email":
		return "string"
	case "bool":
		return "bool"
	case "uuid":
		return "uuid.UUID"
	// TODO: Add more type mappings
	default:
		return "any"
	}
}

func pluralize(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "s"), strings.HasSuffix(lower, "x"), strings.HasSuffix(lower, "z"), strings.HasSuffix(lower, "ch"), strings.HasSuffix(lower, "sh"):
		return name + "es"
	case strings.HasSuffix(lower, "y") && len(name) > 1 && !strings.ContainsRune("aeiou", rune(lower[len(lower)-2])):
		return name[:len(name)-1] + "ies"
	default:
		return name + "s"
	}
}
