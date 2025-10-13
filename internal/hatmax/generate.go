package hatmax

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

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

	// Set output directory - always use examples/
	if outputDir == "." {
		outputDir = filepath.Join("examples", monorepoName)
	}

	fmt.Printf("Generating monorepo '%s' in directory: %s\n", monorepoName, outputDir)

	// Copy the YAML config file to the monorepo root
	if err := copyConfigToMonorepoRoot(usedFile, outputDir); err != nil {
		return fmt.Errorf("error copying config file to monorepo root: %w", err)
	}
	fmt.Printf("Config file %s copied to monorepo root\n", usedFile)

	// Generate monorepo-level core library (bootstrap)
	fmt.Println("Generating monorepo core library...")
	if err := generateMonorepoCoreLibrary(outputDir, config, tmplFS); err != nil {
		return fmt.Errorf("error generating monorepo core library: %w", err)
	}
	fmt.Println("Monorepo core library generated successfully.")

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

		// Calculate core library module path
		if strings.Contains(modulePath, "/services/") {
			// Remove everything from /services/ onwards and add /pkg/lib/core
			baseModulePath := modulePath[:strings.Index(modulePath, "/services/")]
			config.MonorepoModulePath = baseModulePath + "/pkg/lib/core"
		} else {
			// Fallback case
			config.MonorepoModulePath = config.Package + "/pkg/lib/core"
		}

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

		// Core library is generated at monorepo level, not per service

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


	// Final workspace synchronization after all services are generated
	if devMode {
		fmt.Println("Performing final workspace synchronization...")
		if err := finalWorkspaceSync(outputDir); err != nil {
			return fmt.Errorf("error performing final workspace sync: %w", err)
		}
		fmt.Println("Final workspace synchronization completed successfully.")
	}

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

// generateMonorepoCoreLibrary generates the core library at monorepo level
// This creates a shared core library that all services can use
func generateMonorepoCoreLibrary(outputDir string, config Config, tmplFS fs.FS) error {
	// Create the core library directory at monorepo level
	coreDir := filepath.Join(outputDir, "pkg", "lib", "core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		return fmt.Errorf("cannot create core library directory: %w", err)
	}

	// Generate go.mod for the core library module
	if err := generateCoreGoMod(outputDir, config); err != nil {
		return fmt.Errorf("cannot generate core go.mod: %w", err)
	}

	// Generate go.work for the monorepo workspace
	if err := generateMonorepoWorkspace(outputDir, config); err != nil {
		return fmt.Errorf("cannot generate monorepo workspace: %w", err)
	}

	// Get templates filesystem
	templateFS, err := fs.Sub(tmplFS, "assets/templates")
	if err != nil {
		return fmt.Errorf("cannot create templates sub-filesystem: %w", err)
	}

	// Generate each core library file
	coreFileMapping := map[string]string{
		"core_lifecycle.tmpl":  "lifecycle.go",
		"core_server.tmpl":     "server.go",
		"core_log.tmpl":        "log.go",
		"core_auth.tmpl":       "auth.go",
		"core_model.tmpl":      "model.go",
		"core_response.tmpl":   "response.go",
		"core_validation.tmpl": "validation.go",
	}

	// Generate each core library file
	for templateFile, outputFile := range coreFileMapping {
		tmpl, err := template.ParseFS(templateFS, templateFile)
		if err != nil {
			return fmt.Errorf("cannot parse template %s: %w", templateFile, err)
		}

		filePath := filepath.Join(coreDir, outputFile)
		if err := executeTemplate(tmpl, filePath, nil); err != nil {
			return fmt.Errorf("cannot generate core file %s: %w", outputFile, err)
		}

		fmt.Printf("  - Created %s\n", filePath)
	}

	return nil
}

// executeTemplate executes a template and writes it to a file
func executeTemplate(tmpl *template.Template, filePath string, data any) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("cannot create file %s: %w", filePath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("cannot execute template for %s: %w", filePath, err)
	}
	return nil
}

// generateCoreGoMod generates a go.mod file for the core library module
func generateCoreGoMod(outputDir string, config Config) error {
	// Use the package field from config for the core library module
	coreModulePath := "github.com/adrianpk/hatmax-" + filepath.Base(outputDir) + "/pkg/lib/core"
	if config.Package != "" {
		coreModulePath = config.Package + "/pkg/lib/core"
	}
	
	// Create core library go.mod with all necessary dependencies
	goModContent := fmt.Sprintf(`module %s

go 1.23

require (
	github.com/knadh/koanf/v2 v2.3.0
	github.com/knadh/koanf/parsers/yaml v1.1.0
	github.com/knadh/koanf/providers/env v1.1.0
	github.com/knadh/koanf/providers/file v1.0.0
	github.com/knadh/koanf/providers/posflag v1.0.1
	github.com/knadh/koanf/providers/rawbytes v1.0.0
	github.com/spf13/pflag v1.0.10
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
	github.com/mattn/go-sqlite3 v1.14.32
	go.mongodb.org/mongo-driver v1.17.4
)
`, coreModulePath)

	// Write go.mod to the core library directory
	coreDir := filepath.Join(outputDir, "pkg", "lib", "core")
	goModPath := filepath.Join(coreDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		return fmt.Errorf("cannot write core go.mod: %w", err)
	}
	fmt.Printf("  - Created %s\n", goModPath)

	return nil
}

// generateMonorepoWorkspace generates a go.work file for the monorepo workspace
func generateMonorepoWorkspace(outputDir string, config Config) error {
	// Build workspace content dynamically based on services in config
	var workspaceBuilder strings.Builder
	workspaceBuilder.WriteString("go 1.23\n\n")
	workspaceBuilder.WriteString("use (\n")
	workspaceBuilder.WriteString("\t./pkg/lib/core\n") // Include the core library module
	
	// Add all services from config
	for serviceName := range config.Services {
		workspaceBuilder.WriteString(fmt.Sprintf("\t./services/%s\n", serviceName))
	}
	
	workspaceBuilder.WriteString(")\n")

	// Write go.work to the monorepo root
	goWorkPath := filepath.Join(outputDir, "go.work")
	if err := os.WriteFile(goWorkPath, []byte(workspaceBuilder.String()), 0o644); err != nil {
		return fmt.Errorf("cannot write go.work: %w", err)
	}

	fmt.Printf("  - Created %s\n", goWorkPath)
	return nil
}

// finalWorkspaceSync performs final workspace synchronization after all services are generated
func finalWorkspaceSync(outputDir string) error {
	// Get absolute path to avoid issues with relative paths
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("cannot get absolute path for output directory: %w", err)
	}
	
	// Change to the monorepo root directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %w", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()
	
	if err := os.Chdir(absOutputDir); err != nil {
		return fmt.Errorf("cannot change to monorepo root %s: %w", absOutputDir, err)
	}
	
	// Run go mod tidy in monorepo root first
	fmt.Println("  - Running go mod tidy in monorepo root...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: go mod tidy failed in monorepo root: %v\n", err)
	}
	
	// Run go work sync to synchronize all modules
	fmt.Println("  - Running go work sync...")
	cmd = exec.Command("go", "work", "sync")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go work sync failed: %w", err)
	}
	
	return nil
}
