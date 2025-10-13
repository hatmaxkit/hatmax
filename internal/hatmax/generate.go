package hatmax

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func GenerateAction(c *cli.Context, tmplFS fs.FS) error {
	var yamlFile []byte
	var err error
	configFiles := []string{"hatmax.yaml", "hatmax.yml", "monorepo.yaml"}
	var usedFile string

	for _, configFile := range configFiles {
		yamlFile, err = os.ReadFile(configFile)
		if err == nil {
			usedFile = configFile
			break
		}
	}
	if err != nil {
		return fmt.Errorf("error reading config file (tried %v): %w", configFiles, err)
	}

	fmt.Printf("Using config file: %s\n", usedFile)

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return fmt.Errorf("error parsing YAML file: %w", err)
	}

	// Determine base output directory based on mode
	outputDir := c.String("output")
	devMode := c.Bool("dev")

	// Get the monorepo name from config, use "monorepo" as default
	monorepoName := "monorepo"
	if config.Name != "" {
		monorepoName = SanitizeName(config.Name)
	}

	// Set output directory based on mode and name
	if devMode {
		if outputDir == "." {
			outputDir = filepath.Join("examples", monorepoName)
		}
	} else {
		if outputDir == "." {
			outputDir = filepath.Join("generated", monorepoName)
		}
	}

	fmt.Printf("Generating monorepo '%s' in directory: %s\n", monorepoName, outputDir)

	// Copy the YAML config file to the monorepo root
	if err := copyConfigToMonorepoRoot(usedFile, outputDir); err != nil {
		return fmt.Errorf("error copying config file to monorepo root: %w", err)
	}
	fmt.Printf("Config file %s copied to monorepo root\n", usedFile)

	for serviceName := range config.Services {
		servicePath := filepath.Join(outputDir, "services", serviceName)
		modulePath := c.String("module-path")
		if modulePath == "" {
			// Use package field from config if available
			if config.Package != "" {
				modulePath = fmt.Sprintf("%s/services/%s", config.Package, serviceName)
			} else if devMode {
				modulePath = fmt.Sprintf("github.com/adrianpk/hatmax/examples/%s/services/%s", monorepoName, serviceName)
			} else {
				modulePath = inferModulePath(servicePath)
			}
		}
		config.ModulePath = modulePath

		fmt.Printf("Generating service '%s' in '%s'...\n", serviceName, servicePath)
		if err := Scaffold(config, servicePath); err != nil {
			return fmt.Errorf("error generating directories for service %s: %w", serviceName, err)
		}
		fmt.Println("Directory structure generated successfully.")

		modelGen, err := NewModelGenerator(config, servicePath, devMode, tmplFS)
		if err != nil {
			return fmt.Errorf("cannot create model generator for service %s: %w", serviceName, err)
		}

		fmt.Println("Generating config files and XParams...")
		if err := modelGen.GenerateConfigAndXParams(); err != nil {
			return fmt.Errorf("error generating config for service %s: %w", serviceName, err)
		}
		fmt.Println("Config files and XParams generated successfully.")

		fmt.Println("Generating aggregate models...")
		if err := modelGen.GenerateAggregateModels(); err != nil {
			return fmt.Errorf("error generating aggregate models for service %s: %w", serviceName, err)
		}
		fmt.Println("Aggregate models generated successfully.")

		fmt.Println("Generating aggregate repository interfaces...")
		if err := modelGen.GenerateAggregateRepoInterfaces(); err != nil {
			return fmt.Errorf("error generating aggregate repository interfaces for service %s: %w", serviceName, err)
		}
		fmt.Println("Aggregate repository interfaces generated successfully.")

		fmt.Println("Generating models...")
		if err := modelGen.GenerateModels(); err != nil {
			return fmt.Errorf("error generating models for service %s: %w", serviceName, err)
		}
		fmt.Println("Models generated successfully.")

		fmt.Println("Generating repository interfaces...")
		if err := modelGen.GenerateRepoInterfaces(); err != nil {
			return fmt.Errorf("error generating repository interfaces for service %s: %w", serviceName, err)
		}
		fmt.Println("Repository interfaces generated successfully.")

		fmt.Println("Generating service interfaces...")
		if err := modelGen.GenerateServiceInterfaces(); err != nil {
			return fmt.Errorf("error generating service interfaces for service %s: %w", serviceName, err)
		}
		fmt.Println("Service interfaces generated successfully.")

		fmt.Println("Generating SQLite repository implementations...")
		if err := modelGen.GenerateSQLiteRepoImplementations(); err != nil {
			return fmt.Errorf("error generating SQLite repository implementations for service %s: %w", serviceName, err)
		}
		fmt.Println("SQLite repository implementations generated successfully.")

		fmt.Println("Generating SQLite aggregate repository implementations...")
		if err := modelGen.GenerateAggregateSQLiteRepoImplementations(); err != nil {
			return fmt.Errorf("error generating SQLite aggregate repository implementations for service %s: %w", serviceName, err)
		}
		fmt.Println("SQLite aggregate repository implementations generated successfully.")

		fmt.Println("Generating MongoDB repository implementations...")
		if err := modelGen.GenerateMongoRepoImplementations(); err != nil {
			return fmt.Errorf("cannot generate MongoDB repository implementations for service %s: %w", serviceName, err)
		}
		fmt.Println("MongoDB repository implementations generated successfully.")

		fmt.Println("Generating MongoDB aggregate repository implementations...")
		if err := modelGen.GenerateAggregateMongoRepoImplementations(); err != nil {
			return fmt.Errorf("cannot generate MongoDB aggregate repository implementations for service %s: %w", serviceName, err)
		}
		fmt.Println("MongoDB aggregate repository implementations generated successfully.")

		fmt.Println("Generating handlers...")
		if err := modelGen.GenerateHandlers(); err != nil {
			return fmt.Errorf("cannot generate handlers for service %s: %w", serviceName, err)
		}
		fmt.Println("Handlers generated successfully.")

		fmt.Println("Generating validators...")
		if err := modelGen.GenerateValidators(); err != nil {
			return fmt.Errorf("cannot generate validators for service %s: %w", serviceName, err)
		}
		fmt.Println("Validators generated successfully.")

		fmt.Println("Generating main.go...")
		if err := modelGen.GenerateMain(); err != nil {
			return fmt.Errorf("cannot generate main.go for service %s: %w", serviceName, err)
		}
		fmt.Println("main.go generated successfully.")

		fmt.Println("Generating core library files...")
		if err := modelGen.GenerateCoreLibrary(); err != nil {
			return fmt.Errorf("cannot generate core library files for service %s: %w", serviceName, err)
		}
		fmt.Println("Core library files generated successfully.")

		fmt.Println("Generating go.mod...")
		if err := modelGen.GenerateGoMod(); err != nil {
			return fmt.Errorf("cannot generate go.mod for service %s: %w", serviceName, err)
		}
		fmt.Println("go.mod generated successfully.")

		fmt.Println("Running post-generation cleanup...")
		if err := modelGen.PostGenerationCleanup(); err != nil {
			return fmt.Errorf("cannot run post-generation cleanup for service %s: %w", serviceName, err)
		}
		fmt.Println("Post-generation cleanup completed successfully.")

		fmt.Println("Generating Makefile...")
		if err := modelGen.GenerateMakefile(serviceName); err != nil {
			return fmt.Errorf("cannot generate Makefile for service %s: %w", serviceName, err)
		}
		fmt.Println("Makefile generated successfully.")

		fmt.Println("Generating deployment configurations...")
		service := config.Services[serviceName]
		deploymentGen, err := NewDeploymentGenerator(&config, outputDir, serviceName, &service, tmplFS)
		if err != nil {
			return fmt.Errorf("cannot create deployment generator for service %s: %w", serviceName, err)
		}
		if err := deploymentGen.GenerateNomadDeployments(); err != nil {
			return fmt.Errorf("cannot generate deployment configurations for service %s: %w", serviceName, err)
		}
		fmt.Println("Deployment configurations generated successfully.")
	}

	fmt.Println("Generating monorepo-level deployment scripts...")
	if err := generateMonorepoDeploymentScripts(outputDir); err != nil {
		return fmt.Errorf("error generating monorepo deployment scripts: %w", err)
	}
	fmt.Println("Monorepo deployment scripts generated successfully.")

	return nil
}

// copyConfigToMonorepoRoot copies the YAML config file to the monorepo root directory
func copyConfigToMonorepoRoot(configFile, outputDir string) error {
	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	destPath := filepath.Join(outputDir, filepath.Base(configFile))
	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write config file to %s: %w", destPath, err)
	}

	return nil
}

// inferModulePath infers a Go module path based on the output directory
func inferModulePath(outputDir string) string {
	// Convert directory path to a reasonable module name
	// For example: "my-project" -> "github.com/user/my-project"
	// For now, we'll use a simple approach
	if outputDir == "app" {
		return "generatedapp"
	}

	cleanDir := strings.ReplaceAll(outputDir, "/", "-")
	cleanDir = strings.ReplaceAll(cleanDir, "_", "-")
	return cleanDir
}

func generateMonorepoDeploymentScripts(outputDir string) error {
	scriptsDir := filepath.Join(outputDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	deployScript := `#!/bin/bash
set -e

echo "Deploying all services..."

for job in deployments/nomad/jobs/*.nomad; do
    if [ -f "$job" ]; then
        echo "Deploying $(basename "$job")..."
        nomad job run "$job"
    fi
done

echo "All services deployed successfully!"
`

	deployFile := filepath.Join(scriptsDir, "deploy.sh")
	if err := os.WriteFile(deployFile, []byte(deployScript), 0o755); err != nil {
		return fmt.Errorf("failed to write deploy script: %w", err)
	}

	healthScript := `#!/bin/bash

echo "Checking service health..."

for job in deployments/nomad/jobs/*.nomad; do
    service_name=$(basename "$job" .nomad)
    echo "Checking $service_name..."
    
    if nomad job status "$service_name" | grep -q "Status.*running"; then
        echo "✓ $service_name is running"
    else
        echo "✗ $service_name is not running"
    fi
done
`

	healthFile := filepath.Join(scriptsDir, "health-check.sh")
	if err := os.WriteFile(healthFile, []byte(healthScript), 0o755); err != nil {
		return fmt.Errorf("failed to write health check script: %w", err)
	}

	return nil
}
