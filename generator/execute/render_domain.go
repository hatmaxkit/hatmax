package execute

import (
	"fmt"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
)

var domainRenderers = map[string]recipeRenderer{
	"server_rendered_crud.model":                renderModel,
	"server_rendered_crud.store_contract":       renderStoreContract,
	"server_rendered_crud.postgres_store":       renderPostgresStore,
	"server_rendered_crud.service":              renderService,
	"server_rendered_crud.model_tests":          renderModelTests,
	"server_rendered_crud.service_tests":        renderServiceTests,
	"server_rendered_crud.postgres_store_tests": renderPostgresStoreTests,
}

func renderModel(context renderContext, edit Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\n", context.feature)
	source.WriteString("import (\n")

	if context.validation != nil && context.validation.Kind == "pattern" {
		source.WriteString("\t\"regexp\"\n")
	}

	source.WriteString("\t\"time\"\n\n\t\"hatmax.adrianpk.com/model\"\n\t\"hatmax.adrianpk.com/validation\"\n)\n\n")
	fmt.Fprintf(&source, "// %s is the feature-owned domain entity.\ntype %s struct {\n", context.entity, context.entity)
	source.WriteString("\tID string\n")

	for _, field := range context.fields {
		fmt.Fprintf(&source, "\t%s %s\n", field.GoName, field.GoType)
	}

	source.WriteString("\tCreatedAt time.Time\n\tUpdatedAt time.Time\n}\n\n")
	fmt.Fprintf(&source, "// %sInput contains admitted mutable values.\ntype %sInput struct {\n", context.entity, context.entity)

	for _, field := range context.fields {
		fmt.Fprintf(&source, "\t%s %s\n", field.GoName, field.InputType)
	}

	source.WriteString("}\n\n")
	fmt.Fprintf(&source, "// New%s creates a valid %s.\nfunc New%s(input %sInput) (*%s, error) {\n", context.entity, context.entity, context.entity, context.entity, context.entity)
	source.WriteString("\tnow := model.Now()\n\n")
	fmt.Fprintf(&source, "\tvalue := &%s{\n\t\tID: model.NewID(),\n", context.entity)

	for _, field := range context.fields {
		fmt.Fprintf(&source, "\t\t%s: input.%s,\n", field.GoName, field.GoName)
	}

	source.WriteString("\t\tCreatedAt: now,\n\t\tUpdatedAt: now,\n\t}\n\n")
	source.WriteString("\tif err := value.Validate(); err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn value, nil\n}\n\n")
	fmt.Fprintf(&source, "// Update applies validated mutable values.\nfunc (value *%s) Update(input %sInput) error {\n", context.entity, context.entity)

	for _, field := range context.fields {
		fmt.Fprintf(&source, "\tvalue.%s = input.%s\n", field.GoName, field.GoName)
	}

	source.WriteString("\n\tif err := value.Validate(); err != nil {\n\t\treturn err\n\t}\n\n\tvalue.UpdatedAt = model.Now()\n\n\treturn nil\n}\n\n")
	fmt.Fprintf(&source, "// Validate enforces durable %s invariants.\nfunc (value %s) Validate() error {\n", context.entity, context.entity)
	source.WriteString("\tvar errors validation.ValidationErrors\n")

	for _, field := range context.fields {
		renderFieldValidation(&source, field, context.validation)
	}

	source.WriteString("\n\tif errors.HasErrors() {\n\t\treturn errors\n\t}\n\n\treturn nil\n}\n")

	return formatGo(edit.Target, source.String())
}

func renderFieldValidation(source *strings.Builder, field renderField, rule *intent.ValidationRule) {
	customRequired := rule != nil && rule.Field == field.Name && rule.Scope == intent.ValidationDurable && rule.Kind == "required"

	if field.ParseKind == "string" || field.ParseKind == "text" || field.ParseKind == "decimal" || field.ParseKind == "uuid" {
		fmt.Fprintf(source, "\n\t%sValidator := validation.Field(%q, value.%s)", field.Name, field.Name, field.GoName)

		if field.Required && !customRequired {
			source.WriteString(".Required()")
		}

		if field.ParseKind == "string" {
			source.WriteString(".MaxLength(255)")
		}

		fmt.Fprintf(source, "\n\terrors.Merge(%sValidator.Errors())\n", field.Name)
	}

	if field.Required && !customRequired && (field.ParseKind == "date" || field.ParseKind == "timestamp") {
		fmt.Fprintf(source, "\n\tif value.%s.IsZero() {\n\t\terrors.Add(%q, \"is required\")\n\t}\n", field.GoName, field.Name)
	}

	if field.ParseKind == "uuid" {
		fmt.Fprintf(source, "\n\tif value.%s != \"\" {\n\t\tif _, err := model.ParseID(value.%s); err != nil {\n\t\t\terrors.Add(%q, \"must be a valid UUID\")\n\t\t}\n\t}\n", field.GoName, field.GoName, field.Name)
	}

	if rule != nil && rule.Field == field.Name && rule.Scope == intent.ValidationDurable {
		renderAdditionalValidation(source, field, *rule)
	}
}

func renderAdditionalValidation(source *strings.Builder, field renderField, rule intent.ValidationRule) {
	message := validationMessage(rule)

	switch rule.Kind {
	case "required":
		condition := "value." + field.GoName + ` == ""`
		if field.GoType == "time.Time" {
			condition = "value." + field.GoName + ".IsZero()"
		} else if field.GoType == "int64" {
			condition = "value." + field.GoName + " == 0"
		}

		fmt.Fprintf(source, "\n\tif %s {\n\t\terrors.Add(%q, %q)\n\t}\n", condition, field.Name, message)
	case "min_length":
		fmt.Fprintf(source, "\n\tif value.%s != \"\" && len(value.%s) < %s {\n\t\terrors.Add(%q, %q)\n\t}\n", field.GoName, field.GoName, rule.Value, field.Name, message)
	case "max_length":
		fmt.Fprintf(source, "\n\tif len(value.%s) > %s {\n\t\terrors.Add(%q, %q)\n\t}\n", field.GoName, rule.Value, field.Name, message)
	case "minimum":
		fmt.Fprintf(source, "\n\tif value.%s < %s {\n\t\terrors.Add(%q, %q)\n\t}\n", field.GoName, rule.Value, field.Name, message)
	case "maximum":
		fmt.Fprintf(source, "\n\tif value.%s > %s {\n\t\terrors.Add(%q, %q)\n\t}\n", field.GoName, rule.Value, field.Name, message)
	case "pattern":
		fmt.Fprintf(source, "\n\tif value.%s != \"\" && !regexp.MustCompile(%q).MatchString(value.%s) {\n\t\terrors.Add(%q, %q)\n\t}\n", field.GoName, rule.Value, field.GoName, field.Name, message)
	}
}

func validationMessage(rule intent.ValidationRule) string {
	if strings.TrimSpace(rule.Message) != "" {
		return rule.Message
	}

	switch rule.Kind {
	case "maximum":
		return "must be at most " + rule.Value
	case "max_length":
		return "must be at most " + rule.Value + " characters"
	case "minimum":
		return "must be at least " + rule.Value
	case "min_length":
		return "must be at least " + rule.Value + " characters"
	case "pattern":
		return "has an invalid format"
	case "required":
		return "is required"
	default:
		return "is invalid"
	}
}

func renderStoreContract(context renderContext, edit Edit) ([]byte, error) {
	source := fmt.Sprintf(`package %s

import (
	"context"
	"errors"
)

// ErrNotFound classifies a missing %s.
var ErrNotFound = errors.New("%s not found")

// Store is the consumer-owned persistence boundary.
type Store interface {
	List(context.Context) ([]%s, error)
	Get(context.Context, string) (*%s, error)
	Create(context.Context, *%s) error
	Update(context.Context, *%s) error
	Delete(context.Context, string) error
}
`, context.feature, context.feature, context.entity, context.entity, context.entity, context.entity, context.entity)

	return formatGo(edit.Target, source)
}

func renderPostgresStore(context renderContext, edit Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\n", context.feature)
	source.WriteString("import (\n\t\"context\"\n\t\"database/sql\"\n\t\"errors\"\n\t\"fmt\"\n\n")
	fmt.Fprintf(&source, "\t%q\n", context.modulePath+"/internal/dal")
	source.WriteString(")\n\n")
	source.WriteString("// DBProvider supplies the live Hatmax-managed database.\ntype DBProvider interface {\n\tGetDB() *sql.DB\n}\n\n")
	source.WriteString("// PostgresStore persists the feature through generated SQLC queries.\ntype PostgresStore struct {\n\tprovider DBProvider\n\tqueries *dal.Queries\n}\n\n")
	source.WriteString("// NewPostgresStore retains dependencies without performing I/O.\nfunc NewPostgresStore(provider DBProvider) *PostgresStore {\n\treturn &PostgresStore{provider: provider}\n}\n\n")
	source.WriteString("// Start acquires the live database after its provider has started.\nfunc (store *PostgresStore) Start(context.Context) error {\n\tdatabase := store.provider.GetDB()\n\tif database == nil {\n\t\treturn errors.New(\"database connection not available\")\n\t}\n\n\tstore.queries = dal.New(database)\n\n\treturn nil\n}\n\n")
	source.WriteString("// Stop preserves lifecycle alignment.\nfunc (store *PostgresStore) Stop(context.Context) error { return nil }\n\n")
	fmt.Fprintf(&source, "func (store *PostgresStore) List(ctx context.Context) ([]%s, error) {\n\trows, err := store.queries.List%s(ctx)\n", context.entity, context.plural)
	source.WriteString("\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"list records: %w\", err)\n\t}\n\n")
	fmt.Fprintf(&source, "\tresult := make([]%s, 0, len(rows))\n\tfor _, row := range rows {\n\t\tresult = append(result, %sFromRow(row))\n\t}\n\n\treturn result, nil\n}\n\n", context.entity, context.feature)
	fmt.Fprintf(&source, "func (store *PostgresStore) Get(ctx context.Context, id string) (*%s, error) {\n\trow, err := store.queries.Get%s(ctx, id)\n", context.entity, context.entity)
	source.WriteString("\tif errors.Is(err, sql.ErrNoRows) {\n\t\treturn nil, ErrNotFound\n\t}\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"get record: %w\", err)\n\t}\n\n")
	fmt.Fprintf(&source, "\tresult := %sFromRow(row)\n\n\treturn &result, nil\n}\n\n", context.feature)
	fmt.Fprintf(&source, "func (store *PostgresStore) Create(ctx context.Context, value *%s) error {\n\terr := store.queries.Create%s(ctx, dal.Create%sParams{\n", context.entity, context.entity, context.entity)
	renderDALParams(&source, context, "value", true)
	source.WriteString("\t})\n\tif err != nil {\n\t\treturn fmt.Errorf(\"create record: %w\", err)\n\t}\n\n\treturn nil\n}\n\n")
	fmt.Fprintf(&source, "func (store *PostgresStore) Update(ctx context.Context, value *%s) error {\n\tresult, err := store.queries.Update%s(ctx, dal.Update%sParams{\n", context.entity, context.entity, context.entity)
	renderDALParams(&source, context, "value", false)
	source.WriteString("\t})\n\tif err != nil {\n\t\treturn fmt.Errorf(\"update record: %w\", err)\n\t}\n\tif result == 0 {\n\t\treturn ErrNotFound\n\t}\n\n\treturn nil\n}\n\n")
	fmt.Fprintf(&source, "func (store *PostgresStore) Delete(ctx context.Context, id string) error {\n\tresult, err := store.queries.Delete%s(ctx, id)\n", context.entity)
	source.WriteString("\tif err != nil {\n\t\treturn fmt.Errorf(\"delete record: %w\", err)\n\t}\n\tif result == 0 {\n\t\treturn ErrNotFound\n\t}\n\n\treturn nil\n}\n\n")
	fmt.Fprintf(&source, "func %sFromRow(row dal.%s) %s {\n\treturn %s{\n\t\tID: row.ID,\n", context.feature, context.entity, context.entity, context.entity)

	for _, field := range context.fields {
		fmt.Fprintf(&source, "\t\t%s: row.%s,\n", field.GoName, field.GoName)
	}

	source.WriteString("\t\tCreatedAt: row.CreatedAt,\n\t\tUpdatedAt: row.UpdatedAt,\n\t}\n}\n")

	return formatGo(edit.Target, source.String())
}

func renderDALParams(source *strings.Builder, context renderContext, receiver string, includeCreated bool) {
	fmt.Fprintf(source, "\t\tID: %s.ID,\n", receiver)

	for _, field := range context.fields {
		fmt.Fprintf(source, "\t\t%s: %s.%s,\n", field.GoName, receiver, field.GoName)
	}

	if includeCreated {
		fmt.Fprintf(source, "\t\tCreatedAt: %s.CreatedAt,\n", receiver)
	}

	fmt.Fprintf(source, "\t\tUpdatedAt: %s.UpdatedAt,\n", receiver)
}

func renderService(context renderContext, edit Edit) ([]byte, error) {
	source := fmt.Sprintf(`package %s

import (
	"context"
	"fmt"
)

// Service owns %s application workflows.
type Service struct { store Store }

// NewService constructs the feature service.
func NewService(store Store) *Service { return &Service{store: store} }

func (service *Service) List(ctx context.Context) ([]%s, error) {
	values, err := service.store.List(ctx)
	if err != nil { return nil, fmt.Errorf("list %s: %%w", err) }
	return values, nil
}

func (service *Service) Get(ctx context.Context, id string) (*%s, error) {
	value, err := service.store.Get(ctx, id)
	if err != nil { return nil, fmt.Errorf("get %s: %%w", err) }
	return value, nil
}

func (service *Service) Create(ctx context.Context, input %sInput) (*%s, error) {
	value, err := New%s(input)
	if err != nil { return nil, err }
	if err = service.store.Create(ctx, value); err != nil { return nil, fmt.Errorf("create %s: %%w", err) }
	return value, nil
}

func (service *Service) Update(ctx context.Context, id string, input %sInput) (*%s, error) {
	value, err := service.store.Get(ctx, id)
	if err != nil { return nil, fmt.Errorf("get %s: %%w", err) }
	if err = value.Update(input); err != nil { return nil, err }
	if err = service.store.Update(ctx, value); err != nil { return nil, fmt.Errorf("update %s: %%w", err) }
	return value, nil
}

func (service *Service) Delete(ctx context.Context, id string) error {
	if err := service.store.Delete(ctx, id); err != nil { return fmt.Errorf("delete %s: %%w", err) }
	return nil
}
`, context.feature, context.label, context.entity, context.feature, context.entity, context.feature, context.entity, context.entity, context.entity, context.feature, context.entity, context.entity, context.feature, context.feature, context.feature)

	return formatGo(edit.Target, source)
}

func renderModelTests(context renderContext, edit Edit) ([]byte, error) {
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n\nimport (\n\t\"testing\"\n", context.feature)

	if context.validation != nil && (context.validation.Kind == "min_length" || context.validation.Kind == "max_length") {
		source.WriteString("\t\"strings\"\n")
	}

	if fieldsNeedTime(context.fields) {
		source.WriteString("\t\"time\"\n")
	}

	source.WriteString(")\n\n")
	fmt.Fprintf(&source, "func TestNew%sValidatesRequiredFields(t *testing.T) {\n", context.entity)
	fmt.Fprintf(&source, "\ttests := []struct { name string; input %sInput; wantError bool }{\n", context.entity)
	fmt.Fprintf(&source, "\t\t{name: \"valid\", input: %sInput{%s}, wantError: false},\n", context.entity, renderTestInput(context.fields, true))

	if hasRequiredRenderField(context.fields) {
		fmt.Fprintf(&source, "\t\t{name: \"missing required value\", input: %sInput{}, wantError: true},\n", context.entity)
	}

	source.WriteString("\t}\n\n\tfor _, test := range tests {\n\t\tt.Run(test.name, func(t *testing.T) {\n")
	fmt.Fprintf(&source, "\t\t\tvalue, err := New%s(test.input)\n", context.entity)
	source.WriteString("\t\t\tif (err != nil) != test.wantError { t.Fatalf(\"error = %v, wantError %v\", err, test.wantError) }\n\t\t\tif !test.wantError && (value.ID == \"\" || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero()) { t.Error(\"constructor did not establish identity and timestamps\") }\n\t\t})\n\t}\n}\n")

	if context.validation != nil && context.validation.Scope == intent.ValidationDurable && context.validation.Kind != "unique" {
		renderValidationModelTest(&source, context)
	}

	return formatGo(edit.Target, source.String())
}

func renderValidationModelTest(source *strings.Builder, context renderContext) {
	field, found := renderFieldByName(context.fields, context.validation.Field)
	if !found {
		return
	}

	fmt.Fprintf(source, "\nfunc Test%sRejects%sValidation(t *testing.T) {\n", context.entity, exportedName(context.validation.Field))
	fmt.Fprintf(source, "\tinput := %sInput{%s}\n", context.entity, renderTestInput(context.fields, true))
	fmt.Fprintf(source, "\tinput.%s = %s\n", field.GoName, invalidValidationValue(field, *context.validation))
	fmt.Fprintf(source, "\tif _, err := New%s(input); err == nil { t.Fatal(\"New%s() error = nil, want validation failure\") }\n}\n", context.entity, context.entity)
}

func renderFieldByName(fields []renderField, name string) (renderField, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}

	return renderField{}, false
}

func invalidValidationValue(field renderField, rule intent.ValidationRule) string {
	switch rule.Kind {
	case "required":
		return zeroValue(field)
	case "minimum":
		return rule.Value + " - 1"
	case "maximum":
		return rule.Value + " + 1"
	case "min_length":
		return fmt.Sprintf("strings.Repeat(\"x\", %s-1)", rule.Value)
	case "max_length":
		return fmt.Sprintf("strings.Repeat(\"x\", %s+1)", rule.Value)
	case "pattern":
		return `"invalid"`
	default:
		return testValue(field, false)
	}
}

func zeroValue(field renderField) string {
	switch field.GoType {
	case "bool":
		return "false"
	case "int64":
		return "0"
	case "time.Time":
		return "time.Time{}"
	default:
		return `""`
	}
}

func renderServiceTests(context renderContext, edit Edit) ([]byte, error) {
	imports := ""
	if fieldsNeedTime(context.fields) {
		imports = "\n\t\"time\""
	}

	source := fmt.Sprintf(`package %s

import (
	"context"
	"testing"%s
)

type recordingStore struct { created *%s }
func (store *recordingStore) List(context.Context) ([]%s, error) { return nil, nil }
func (store *recordingStore) Get(context.Context, string) (*%s, error) { return nil, ErrNotFound }
func (store *recordingStore) Create(_ context.Context, value *%s) error { store.created = value; return nil }
func (store *recordingStore) Update(context.Context, *%s) error { return nil }
func (store *recordingStore) Delete(context.Context, string) error { return nil }

func TestServiceCreateValidatesBeforePersistence(t *testing.T) {
	store := &recordingStore{}
	service := NewService(store)
	_, err := service.Create(context.Background(), %sInput{%s})
	if err != nil { t.Fatalf("Create() error = %%v", err) }
	if store.created == nil { t.Fatal("Create() did not persist the valid entity") }
}
`, context.feature, imports, context.entity, context.entity, context.entity, context.entity, context.entity, context.entity, renderTestInput(context.fields, true))

	return formatGo(edit.Target, source)
}

func renderPostgresStoreTests(context renderContext, edit Edit) ([]byte, error) {
	imports := ""
	if fieldsNeedTime(context.fields) {
		imports = "\n\t\"time\""
	}

	source := fmt.Sprintf(`package %s

import (
	"context"
	"database/sql"
	"errors"
	"testing"%s

	"hatmax.adrianpk.com/testhelper"
)

type testDBProvider struct { database *sql.DB }
func (provider testDBProvider) GetDB() *sql.DB { return provider.database }

func TestPostgresStoreRequiresStartedDatabase(t *testing.T) {
	store := NewPostgresStore(testDBProvider{})
	if err := store.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want unavailable database")
	}
}

func TestPostgresStoreCRUD(t *testing.T) {
	database, _, cleanup := testhelper.SetupTestDB(t)
	defer cleanup()

	_, err := database.Exec(%q)
	if err != nil { t.Fatalf("create table: %%v", err) }

	store := NewPostgresStore(testDBProvider{database: database})
	if err = store.Start(context.Background()); err != nil { t.Fatalf("Start() error = %%v", err) }

	value, err := New%s(%sInput{%s})
	if err != nil { t.Fatalf("New%s() error = %%v", err) }
	if err = store.Create(context.Background(), value); err != nil { t.Fatalf("Create() error = %%v", err) }

	loaded, err := store.Get(context.Background(), value.ID)
	if err != nil || loaded.ID != value.ID { t.Fatalf("Get() = %%#v, %%v", loaded, err) }

	values, err := store.List(context.Background())
	if err != nil || len(values) != 1 { t.Fatalf("List() = %%#v, %%v", values, err) }

	if err = store.Delete(context.Background(), value.ID); err != nil { t.Fatalf("Delete() error = %%v", err) }
	if _, err = store.Get(context.Background(), value.ID); !errors.Is(err, ErrNotFound) { t.Fatalf("Get() after Delete() error = %%v", err) }
}
`, context.feature, imports, renderTestTableSQL(context), context.entity, context.entity, renderTestInput(context.fields, true), context.entity)

	return formatGo(edit.Target, source)
}

func renderTestTableSQL(context renderContext) string {
	var source strings.Builder
	fmt.Fprintf(&source, "CREATE TABLE %s (id TEXT PRIMARY KEY, ", context.table)

	for _, field := range context.fields {
		fmt.Fprintf(&source, "%s %s NOT NULL, ", field.Name, field.SQLType)
	}

	source.WriteString("created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL)")

	return source.String()
}

func renderTestInput(fields []renderField, valid bool) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		value := testValue(field, valid)
		if value != "" {
			values = append(values, field.GoName+": "+value)
		}
	}

	return strings.Join(values, ", ")
}

func testValue(field renderField, valid bool) string {
	if !valid {
		return ""
	}

	switch field.ParseKind {
	case "boolean":
		return "true"
	case "date", "timestamp":
		return "time.Unix(1, 0)"
	case "integer":
		return "1"
	case "uuid":
		return `"550e8400-e29b-41d4-a716-446655440000"`
	default:
		return `"value"`
	}
}

func hasRequiredRenderField(fields []renderField) bool {
	for _, field := range fields {
		if field.Required && field.ParseKind != "boolean" {
			return true
		}
	}

	return false
}

func fieldsNeedTime(fields []renderField) bool {
	for _, field := range fields {
		if field.ParseKind == "date" || field.ParseKind == "timestamp" {
			return true
		}
	}

	return false
}
