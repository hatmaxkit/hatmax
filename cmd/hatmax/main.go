package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hatmax.adrianpk.com/internal/hatmaxcli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app, err := hatmaxcli.New(hatmaxcli.Config{
		Input:              os.Stdin,
		Output:             os.Stdout,
		ErrorOutput:        os.Stderr,
		WorkingDirectory:   os.Getwd,
		CoordinatorFactory: hatmaxcli.NewCodexCoordinator,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hatmax: initialize command: %v\n", err)

		os.Exit(hatmaxcli.ExitFailure)
	}

	os.Exit(app.Run(ctx, os.Args[1:]))
}
