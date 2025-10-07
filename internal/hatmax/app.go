package hatmax

import (
	"io/fs"

	"github.com/urfave/cli/v2"
)

func NewApp(templateFS fs.FS) *cli.App {
	app := &cli.App{
		Name:  "hatmax",
		Usage: "A Go-based monorepo generator",
		Commands: []*cli.Command{
			{
				Name:    "generate",
				Aliases: []string{"g"},
				Usage:   "Generate the directory structure based on monorepo.yaml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output directory for generated code",
						Value:   ".",
					},
					&cli.StringFlag{
						Name:    "module-path",
						Aliases: []string{"m"},
						Usage:   "Go module path for generated code (auto-inferred if not specified)",
					},
					&cli.BoolFlag{
						Name:  "dev",
						Usage: "Enable development mode",
					},
				},
				Action: func(c *cli.Context) error {
					return GenerateAction(c, templateFS)
				},
			},
		},
	}
	return app
}
