package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/internal/hatmaxcli"
	"hatmax.adrianpk.com/internal/hatmaxtui"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) != 1 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: hm")

		os.Exit(2)
	}

	root, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hm: determine project directory: %v\n", err)

		os.Exit(1)
	}

	runtime, err := hatmaxcli.NewCodexConversationRuntime(root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hm: initialize conversation: %v\n", err)

		os.Exit(1)
	}

	err = hatmaxtui.Run(ctx, hatmaxtui.Config{
		Input:       os.Stdin,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Root:        root,
		Sessions: func(
			openContext context.Context,
			openRoot string,
			options conversation.SessionOptions,
		) (hatmaxtui.Session, error) {
			return runtime.Open(openContext, openRoot, options)
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hm: %v\n", err)

		os.Exit(1)
	}
}
