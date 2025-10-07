package hatmax

import (
	"fmt"
	"os"
	"path/filepath"
)

func Scaffold(config Config, outputDir string) error {
	for serviceName, service := range config.Services {
		fmt.Printf("  - Scaffolding service: %s\n", serviceName)

		featureDir := filepath.Join(outputDir, "internal", serviceName)
		if err := os.MkdirAll(featureDir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", featureDir, err)
		}
		fmt.Printf("    - Created %s\n", featureDir)

		for _, repoImpl := range service.RepoImpl {
			repoDir := filepath.Join(outputDir, "internal", repoImpl)
			if err := os.MkdirAll(repoDir, 0o755); err != nil {
				return fmt.Errorf("cannot create directory %s: %w", repoDir, err)
			}
			fmt.Printf("    - Created %s\n", repoDir)
		}
	}
	return nil
}
