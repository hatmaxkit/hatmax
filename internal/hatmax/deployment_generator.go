package hatmax

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type DeploymentGenerator struct {
	Config         *Config
	OutputDir      string
	ServiceName    string
	Service        *Service
	TemplateFS     fs.FS
	JobTemplate    *template.Template
	ConfigTemplate *template.Template
}

type NomadJobData struct {
	ServiceName         string
	Datacenter          string
	Port                int
	Replicas            int
	Resources           *ResourceConfig
	HealthCheckPath     string
	HealthCheckInterval string
	TraefikRule         string
	TraefikEntrypoint   string
	ConsulTags          []string
	ConsulAddress       string
	ConfigTemplate      string
}

type NomadConfigData struct {
	ServiceName   string
	DatabaseType  string
	DatabaseDSN   string
	ConsulAddress string
}

func NewDeploymentGenerator(config *Config, outputDir, serviceName string, service *Service, templateFS fs.FS) (*DeploymentGenerator, error) {
	dg := &DeploymentGenerator{
		Config:      config,
		OutputDir:   outputDir,
		ServiceName: serviceName,
		Service:     service,
		TemplateFS:  templateFS,
	}

	if err := dg.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load deployment templates: %w", err)
	}

	return dg, nil
}

func (dg *DeploymentGenerator) loadTemplates() error {
	jobTemplateContent, err := fs.ReadFile(dg.TemplateFS, "assets/templates/deployment/nomad/job.nomad.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read job template: %w", err)
	}

	dg.JobTemplate, err = template.New("nomad_job").Parse(string(jobTemplateContent))
	if err != nil {
		return fmt.Errorf("failed to parse job template: %w", err)
	}

	configTemplateContent, err := fs.ReadFile(dg.TemplateFS, "assets/templates/deployment/nomad/config.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read config template: %w", err)
	}

	dg.ConfigTemplate, err = template.New("nomad_config").Parse(string(configTemplateContent))
	if err != nil {
		return fmt.Errorf("failed to parse config template: %w", err)
	}

	return nil
}

func (dg *DeploymentGenerator) GenerateNomadDeployments() error {
	if dg.Config.Deployment == nil || dg.Config.Deployment.Nomad == nil {
		return nil
	}

	if dg.Service.Deployment == nil || dg.Service.Deployment.Nomad == nil {
		return nil
	}

	deploymentDir := filepath.Join(dg.OutputDir, "deployments", "nomad", "jobs")
	if err := os.MkdirAll(deploymentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	jobData := dg.buildJobData()
	configTemplate := dg.generateConfigTemplate()

	jobData.ConfigTemplate = configTemplate

	if err := dg.renderJobFile(deploymentDir, jobData); err != nil {
		return fmt.Errorf("failed to render job file: %w", err)
	}

	return nil
}

func (dg *DeploymentGenerator) buildJobData() *NomadJobData {
	serviceConfig := dg.Service.Deployment.Nomad
	globalConfig := dg.Config.Deployment.Nomad

	resources := serviceConfig.Resources
	if resources == nil {
		resources = globalConfig.DefaultResources
	}

	consulAddress := "127.0.0.1:8500"
	if dg.Config.Deployment.Infrastructure != nil && dg.Config.Deployment.Infrastructure.Consul != nil {
		consulAddress = dg.Config.Deployment.Infrastructure.Consul.Address
	}

	entrypoint := "web"
	if dg.Config.Deployment.Infrastructure != nil && dg.Config.Deployment.Infrastructure.Traefik != nil {
		entrypoint = dg.Config.Deployment.Infrastructure.Traefik.Entrypoint
	}

	consulTags := []string{}
	if serviceConfig.Consul != nil {
		consulTags = serviceConfig.Consul.Tags
	}

	return &NomadJobData{
		ServiceName:         dg.ServiceName,
		Datacenter:          globalConfig.Datacenter,
		Port:                serviceConfig.Port,
		Replicas:            serviceConfig.Replicas,
		Resources:           resources,
		HealthCheckPath:     serviceConfig.HealthCheck.Path,
		HealthCheckInterval: serviceConfig.HealthCheck.Interval,
		TraefikRule:         serviceConfig.Traefik.Rule,
		TraefikEntrypoint:   entrypoint,
		ConsulTags:          consulTags,
		ConsulAddress:       consulAddress,
	}
}

func (dg *DeploymentGenerator) generateConfigTemplate() string {
	consulAddress := "127.0.0.1:8500"
	if dg.Config.Deployment.Infrastructure != nil && dg.Config.Deployment.Infrastructure.Consul != nil {
		consulAddress = dg.Config.Deployment.Infrastructure.Consul.Address
	}

	primaryRepo := dg.Service.RepoImpl[0]
	databaseType := strings.ToLower(primaryRepo)

	var databaseDSN string
	switch databaseType {
	case "sqlite":
		databaseDSN = fmt.Sprintf("/alloc/data/%s.db", dg.ServiceName)
	case "mongo":
		databaseDSN = "{{`{{ env \"MONGO_URI\" | default \"mongodb://localhost:27017\" }}`}}"
	default:
		databaseDSN = fmt.Sprintf("/alloc/data/%s.db", dg.ServiceName)
	}

	configData := &NomadConfigData{
		ServiceName:   dg.ServiceName,
		DatabaseType:  databaseType,
		DatabaseDSN:   databaseDSN,
		ConsulAddress: consulAddress,
	}

	var buf bytes.Buffer
	if err := dg.ConfigTemplate.Execute(&buf, configData); err != nil {
		return fmt.Sprintf("# Error generating config: %v", err)
	}

	return buf.String()
}

func (dg *DeploymentGenerator) renderJobFile(outputDir string, jobData *NomadJobData) error {
	var buf bytes.Buffer
	if err := dg.JobTemplate.Execute(&buf, jobData); err != nil {
		return fmt.Errorf("failed to execute job template: %w", err)
	}

	jobFile := filepath.Join(outputDir, fmt.Sprintf("%s.nomad", dg.ServiceName))
	if err := os.WriteFile(jobFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write job file: %w", err)
	}

	return nil
}
