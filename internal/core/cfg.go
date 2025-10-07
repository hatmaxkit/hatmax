package core

type Config struct {
	Workspace *WorkspaceConfig
}

type WorkspaceConfig struct {
	DevMode    bool
	OutputDir  string
	ModulePath string
}

func NewConfig() Config {
	return Config{
		Workspace: &WorkspaceConfig{
			DevMode:   false,
			OutputDir: ".",
		},
	}
}

func BuildConfig(devMode bool, outputDir string, modulePath string) Config {
	if outputDir == "" {
		if devMode {
			outputDir = "example/ref"
		} else {
			outputDir = "."
		}
	}
	return Config{
		Workspace: &WorkspaceConfig{
			DevMode:    devMode,
			OutputDir:  outputDir,
			ModulePath: modulePath,
		},
	}
}
