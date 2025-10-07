# Deployment Generation

**Status:** Draft | **Updated:** 2025-10-07 | **Version:** 0.1

## Overview

Automated generation of deployment descriptors for different orchestration platforms from the monorepo configuration. While the system should eventually support Docker Compose, Kubernetes, and Nomad, the initial focus is on Nomad job definitions with Consul integration.

## Current Implementation

**Status**: Not implemented  
**Location**: N/A  
**Features**: No deployment generation exists yet

## Planned Implementation

### Supported Platforms

**Phase 1: Nomad Focus**
- Nomad job definitions with service discovery
- Consul integration for configuration
- Traefik integration for routing
- Health checks and resource allocation

**Future Phases**:
- Docker Compose for development
- Kubernetes manifests for cloud deployment
- Terraform modules for infrastructure

### Directory Structure

Generated monorepo deployment organization:
```
examples/ref/                     # Generated monorepo root
├── services/
│   ├── todo/
│   └── auth/
├── deployments/                  # Generated deployment configs
│   └── nomad/                   # Platform-specific configs
│       ├── jobs/                # Job definitions per service
│       │   ├── todo.nomad
│       │   └── auth.nomad
│       ├── policies/            # Access policies
│       │   └── service-policies.hcl
│       └── configs/             # Infrastructure configs
│           ├── consul.hcl
│           └── traefik.nomad
└── scripts/                     # Deployment helpers
    ├── deploy.sh
    └── health-check.sh
```

### YAML Configuration Extension

Extension to `hatmax.yml` to support deployment configuration:

```yaml
version: 0.2
name: ref

deployment:
  platforms: [nomad]              # Supported: nomad, docker-compose, kubernetes
  
  nomad:
    datacenter: dc1
    consul_integration: true
    traefik_integration: true
    default_resources:
      cpu: 256
      memory: 128
    
  infrastructure:
    consul:
      enabled: true
      address: "127.0.0.1:8500"
    traefik:
      enabled: true
      entrypoint: web
      domain: "localhost"

services:
  todo:
    kind: domain
    deployment:
      nomad:
        port: 8080
        replicas: 1
        resources:
          cpu: 256
          memory: 128
        health_check:
          path: "/health"
          interval: "30s"
        traefik:
          rule: "PathPrefix(`/todo`)"
          priority: 100
        consul:
          service_name: "todo"
          tags: ["api", "v1"]
```

## Generated Nomad Job Template

**Core job structure** for each service:

```hcl
job "{{.ServiceName}}" {
  datacenters = ["{{.Datacenter}}"]
  type        = "service"

  group "{{.ServiceName}}" {
    count = {{.Replicas}}

    network {
      port "http" {
        static = {{.Port}}
      }
    }

    service {
      name = "{{.ServiceName}}"
      port = "http"
      
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.{{.ServiceName}}.rule={{.TraefikRule}}",
        "traefik.http.routers.{{.ServiceName}}.entrypoints={{.TraefikEntrypoint}}",
        {{range .ConsulTags}}"{{.}}",{{end}}
      ]

      check {
        type     = "http"
        path     = "{{.HealthCheckPath}}"
        interval = "{{.HealthCheckInterval}}"
        timeout  = "5s"
      }
    }

    task "{{.ServiceName}}" {
      driver = "exec"

      config {
        command = "./{{.ServiceName}}"
        args    = ["--config", "/local/config.yaml"]
      }

      template {
        data = <<EOH
{{.ConfigTemplate}}
EOH
        destination = "local/config.yaml"
        change_mode = "restart"
      }

      resources {
        cpu    = {{.Resources.CPU}}
        memory = {{.Resources.Memory}}
      }

      env {
        SERVICE_NAME = "{{.ServiceName}}"
        LOG_LEVEL    = "info"
        CONSUL_HTTP_ADDR = "{{ env "CONSUL_HTTP_ADDR" | default "127.0.0.1:8500" }}"
      }
    }
  }
}
```

### Configuration Template Generation

**Service-specific config template**:

```yaml
# Generated config template for {{.ServiceName}}
server:
  port: {{ env "NOMAD_PORT_http" }}
  host: "0.0.0.0"

database:
  type: "{{.Database.Type}}"
  dsn: "{{.Database.DSN}}"

logging:
  level: "info"
  format: "json"

observability:
  metrics:
    enabled: true
    endpoint: "/metrics"
  tracing:
    enabled: {{.Tracing.Enabled}}

consul:
  address: "{{ env "CONSUL_HTTP_ADDR" | default "127.0.0.1:8500" }}"
  service_name: "{{.ServiceName}}"
```

## Generator Implementation

### Template Organization

**Template structure**:
```
assets/templates/deployment/
├── nomad/
│   ├── job.nomad.tmpl          # Main job template
│   ├── config.yaml.tmpl        # Service configuration
│   └── policy.hcl.tmpl         # Consul policies
├── docker-compose/             # Future
│   └── service.yml.tmpl
└── kubernetes/                 # Future
    ├── deployment.yaml.tmpl
    └── service.yaml.tmpl
```

### Generation Logic

**DeploymentGenerator responsibilities**:

```go
type DeploymentGenerator struct {
    Config         *Config
    OutputDir      string
    JobTemplate    *template.Template
    ConfigTemplate *template.Template
}

func (dg *DeploymentGenerator) GenerateNomadJobs() error {
    for serviceName, service := range dg.Config.Services {
        jobData := &NomadJobData{
            ServiceName:         serviceName,
            Datacenter:          dg.Config.Deployment.Nomad.Datacenter,
            Port:                service.Deployment.Nomad.Port,
            Replicas:           service.Deployment.Nomad.Replicas,
            Resources:          service.Deployment.Nomad.Resources,
            HealthCheckPath:    service.Deployment.Nomad.HealthCheck.Path,
            TraefikRule:        service.Deployment.Nomad.Traefik.Rule,
            ConfigTemplate:     dg.generateConfigTemplate(service),
        }
        
        if err := dg.renderJobFile(serviceName, jobData); err != nil {
            return err
        }
    }
    return nil
}
```

## Planned Features

### Infrastructure Services
- **Status**: Planned
- **Description**: Generate Consul, Traefik, and Vault job definitions
- **Benefits**: Complete infrastructure-as-code from single configuration

### Multi-Environment Support
- **Status**: Planned  
- **Description**: Dev, staging, prod environment variants
- **Implementation**: Environment-specific overlays in deployment config

### Service Dependencies
- **Status**: Planned
- **Description**: Declare service dependencies for startup ordering
- **Implementation**: Nomad job constraints and service discovery health checks

### Secrets Management
- **Status**: Planned
- **Description**: Vault integration for sensitive configuration
- **Implementation**: Vault templates in Nomad jobs

## Next Steps

### Immediate
1. **Extend YAML schema** with deployment configuration
2. **Create Nomad job templates** with basic service definition
3. **Implement DeploymentGenerator** with template rendering

### Medium Term
4. **Add infrastructure service generation** (Consul, Traefik)
5. **Implement environment-specific configurations**
6. **Add deployment scripts** and health check utilities

---

**Summary**: Generator extension to produce deployment descriptors from monorepo configuration, starting with Nomad jobs and focusing on service discovery, load balancing, and configuration management integration.