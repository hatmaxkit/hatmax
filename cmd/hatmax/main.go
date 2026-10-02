// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

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

	if len(os.Args) > 1 {
		os.Exit(runHeadless(ctx, os.Args[1:]))
	}

	root, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hatmax: determine project directory: %v\n", err)

		os.Exit(hatmaxcli.ExitFailure)
	}

	runtime, err := hatmaxcli.NewCodexConversationRuntime(root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hatmax: initialize conversation: %v\n", err)

		os.Exit(hatmaxcli.ExitFailure)
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
		_, _ = fmt.Fprintf(os.Stderr, "hatmax: %v\n", err)

		os.Exit(hatmaxcli.ExitFailure)
	}
}

func runHeadless(ctx context.Context, arguments []string) int {
	app, err := hatmaxcli.New(hatmaxcli.Config{
		Input:              os.Stdin,
		Output:             os.Stdout,
		ErrorOutput:        os.Stderr,
		WorkingDirectory:   os.Getwd,
		CoordinatorFactory: hatmaxcli.NewCodexCoordinator,
		ConversationFactory: func(root string) (hatmaxcli.ConversationManager, error) {
			return hatmaxcli.NewCodexConversationRuntime(root)
		},
		CommandName: "hatmax",
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "hatmax: initialize command: %v\n", err)

		return hatmaxcli.ExitFailure
	}

	return app.Run(ctx, arguments)
}
