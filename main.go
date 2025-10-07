package main

import (
	"embed"
	"log"
	"os"

	"github.com/adrianpk/hatmax/internal/hatmax"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	app := hatmax.NewApp(assetsFS)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
