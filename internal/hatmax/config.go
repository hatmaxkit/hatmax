package hatmax

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level structure of the monorepo.yaml file.
type Config struct {
	Version    string             `yaml:"version"`
	Name       string             `yaml:"name,omitempty"`
	Package    string             `yaml:"package,omitempty"`
	ModulePath string             `yaml:"module_path,omitempty"`
	Deployment *DeploymentConfig  `yaml:"deployment,omitempty"`
	Services   map[string]Service `yaml:"services"`
}

// Service defines a microservice within the monorepo.
type Service struct {
	Kind       string                   `yaml:"kind"`
	RepoImpl   StringOrSlice            `yaml:"repo_impl"`
	Auth       *AuthConfig              `yaml:"auth,omitempty"`
	Deployment *ServiceDeploymentConfig `yaml:"deployment,omitempty"`
	Models     map[string]Model         `yaml:"models"`
	Aggregates map[string]AggregateRoot `yaml:"aggregates,omitempty"`
	API        *APIConfig               `yaml:"api"`
}

// AggregateRoot defines an aggregate root in the domain.
type AggregateRoot struct {
	Table        string                     `yaml:"table"`
	ID           string                     `yaml:"id"`
	VersionField string                     `yaml:"version_field"`
	Fields       map[string]Field           `yaml:"fields"`
	Audit        bool                       `yaml:"audit"`
	SoftDelete   bool                       `yaml:"soft_delete"`
	Children     map[string]ChildCollection `yaml:"children,omitempty"`
}

// ChildCollection defines a child collection within an aggregate root.
type ChildCollection struct {
	Of          string            `yaml:"of"`
	Table       string            `yaml:"table"`
	FK          ForeignKey        `yaml:"fk"`
	ID          string            `yaml:"id"`
	Order       *Order            `yaml:"order,omitempty"`
	Updatable   []string          `yaml:"updatable,omitempty"`
	Audit       bool              `yaml:"audit"`
	Constraints *ChildConstraints `yaml:"constraints,omitempty"`
}

// ForeignKey defines the foreign key relationship from child to root.
type ForeignKey struct {
	Name     string `yaml:"name"`
	Ref      string `yaml:"ref"`
	OnDelete string `yaml:"on_delete"` // restrict|cascade
}

// Order defines ordering for a child collection.
type Order struct {
	Field       string   `yaml:"field"`
	UniqueScope []string `yaml:"unique_scope,omitempty"`
}

// ChildConstraints defines additional constraints for a child collection.
type ChildConstraints struct {
	Unique  [][]string `yaml:"unique,omitempty"`
	Indexes [][]string `yaml:"indexes,omitempty"`
}

// Model defines a domain model.
type Model struct {
	Fields  map[string]Field `yaml:"fields"`
	Options *ModelOptions    `yaml:"options,omitempty"`
}

// ModelOptions defines optional settings for a model.
type ModelOptions struct {
	Audit     bool     `yaml:"audit"`
	Lifecycle []string `yaml:"lifecycle,omitempty"`
}

// Field defines a field within a model or aggregate.
type Field struct {
	Type        string           `yaml:"type"`
	Validations []ValidationRule `yaml:"validations,omitempty"`
	Default     any              `yaml:"default,omitempty"`
}

// ValidationRule defines a validation rule for a field.
type ValidationRule struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value,omitempty"`
}

// APIConfig defines API-related settings for a service.
type APIConfig struct {
	BasePath string    `yaml:"base_path"`
	Handlers []Handler `yaml:"handlers"`
}

// Handler defines an API handler.
type Handler struct {
	ID              string            `yaml:"id"`
	Route           string            `yaml:"route"`
	Source          HandlerSource     `yaml:"source"`
	Model           string            `yaml:"model"`
	Operation       StandardOp        `yaml:"op"`
	CustomOperation string            `yaml:"custom_operation,omitempty"`
	Overrides       *HandlerOverrides `yaml:"overrides,omitempty"`
}

// HandlerSource defines where the handler logic comes from.
type HandlerSource string

const (
	RepoHandlerSource    HandlerSource = "repo"
	ServiceHandlerSource HandlerSource = "service"
	UsecaseHandlerSource HandlerSource = "usecase"
)

// StandardOp defines standard CRUD operations.
type StandardOp string

const (
	OpCreate StandardOp = "create"
	OpGet    StandardOp = "get"
	OpList   StandardOp = "list"
	OpUpdate StandardOp = "update"
	OpDelete StandardOp = "delete"
	OpCustom StandardOp = "custom"
)

// AuthConfig defines authentication and authorization settings.
type AuthConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Mode            string   `yaml:"mode,omitempty"`
	RequiredScopes  []string `yaml:"required_scopes,omitempty"`
	IdentityService string   `yaml:"identity_service,omitempty"`
	CacheTTL        string   `yaml:"cache_ttl,omitempty"`
}

// HandlerOverrides allows overriding generated handler names.
type HandlerOverrides struct {
	RepoName    string `yaml:"repo_name,omitempty"`
	MethodName  string `yaml:"method_name,omitempty"`
	HandlerName string `yaml:"handler_name,omitempty"`
}

// StringOrSlice is a custom type to unmarshal a single string or a slice of strings.
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var str string
		if err := value.Decode(&str); err != nil {
			return fmt.Errorf("cannot decode scalar into string: %w", err)
		}
		*s = []string{str}
		return nil
	}
	if value.Kind == yaml.SequenceNode {
		var slice []string
		if err := value.Decode(&slice); err != nil {
			return fmt.Errorf("cannot unmarshal repo_impl into string or slice: %w", err)
		}
		*s = slice
		return nil
	}
	return fmt.Errorf("cannot unmarshal repo_impl into string or slice: unexpected YAML kind %v", value.Kind)
}

// InferRepoName infers the repository name from the model name.
func (h *Handler) InferRepoName() string {
	return h.Model + "Repo"
}

// InferMethodName infers the method name from the operation.
func (h *Handler) InferMethodName() string {
	switch h.Operation {
	case OpCreate:
		return "Create"
	case OpGet:
		return "Get"
	case OpList:
		return "List"
	case OpUpdate:
		return "Update"
	case OpDelete:
		return "Delete"
	case OpCustom:
		if h.CustomOperation != "" {
			return capitalizeFirst(h.CustomOperation)
		}
		return "CustomOperation"
	default:
		return "UnknownOperation"
	}
}

// InferHandlerName infers the handler function name.
func (h *Handler) InferHandlerName() string {
	return h.InferMethodName() + h.Model
}

// InferRepoCall infers the repository method call.
func (h *Handler) InferRepoCall() string {
	return fmt.Sprintf("h.repo.%s(ctx, item)", h.InferMethodName())
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// SanitizeName sanitizes a name to be filesystem and URL-safe.
// It converts to lowercase, replaces spaces and hyphens with underscores,
// and removes any non-alphanumeric characters except underscores.
func SanitizeName(name string) string {
	if name == "" {
		return name
	}

	// Convert to lowercase
	result := strings.ToLower(name)

	// Replace spaces and hyphens with underscores
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "-", "_")

	// Remove any character that is not alphanumeric or underscore
	var sanitized strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized.WriteRune(r)
		}
	}

	return sanitized.String()
}

type DeploymentConfig struct {
	Platforms      []string              `yaml:"platforms"`
	Nomad          *NomadConfig          `yaml:"nomad,omitempty"`
	Infrastructure *InfrastructureConfig `yaml:"infrastructure,omitempty"`
}

type NomadConfig struct {
	Datacenter         string          `yaml:"datacenter"`
	ConsulIntegration  bool            `yaml:"consul_integration"`
	TraefikIntegration bool            `yaml:"traefik_integration"`
	DefaultResources   *ResourceConfig `yaml:"default_resources,omitempty"`
}

type InfrastructureConfig struct {
	Consul  *ConsulConfig  `yaml:"consul,omitempty"`
	Traefik *TraefikConfig `yaml:"traefik,omitempty"`
}

type ConsulConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type TraefikConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Entrypoint string `yaml:"entrypoint"`
	Domain     string `yaml:"domain"`
}

type ServiceDeploymentConfig struct {
	Nomad *NomadServiceConfig `yaml:"nomad,omitempty"`
}

type NomadServiceConfig struct {
	Port        int                   `yaml:"port"`
	Replicas    int                   `yaml:"replicas"`
	Resources   *ResourceConfig       `yaml:"resources,omitempty"`
	HealthCheck *HealthCheckConfig    `yaml:"health_check,omitempty"`
	Traefik     *ServiceTraefikConfig `yaml:"traefik,omitempty"`
	Consul      *ServiceConsulConfig  `yaml:"consul,omitempty"`
}

type ResourceConfig struct {
	CPU    int `yaml:"cpu"`
	Memory int `yaml:"memory"`
}

type HealthCheckConfig struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
}

type ServiceTraefikConfig struct {
	Rule     string `yaml:"rule"`
	Priority int    `yaml:"priority"`
}

type ServiceConsulConfig struct {
	ServiceName string   `yaml:"service_name"`
	Tags        []string `yaml:"tags,omitempty"`
}
