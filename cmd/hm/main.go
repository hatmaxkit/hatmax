package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hatmax.adrianpk.com/internal/hatmaxtui"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) != 1 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: hm")

		os.Exit(2)
	}

	err := hatmaxtui.Run(ctx, hatmaxtui.Config{
		Input:       os.Stdin,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hm: %v\n", err)

		os.Exit(1)
	}
}
