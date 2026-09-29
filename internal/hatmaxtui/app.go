// Package hatmaxtui implements the terminal presentation adapter for Hatmax.
package hatmaxtui

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
)

// Config supplies the terminal streams owned by the TUI process.
type Config struct {
	Input       io.Reader
	Output      io.Writer
	ErrorOutput io.Writer
}

// Run owns the Bubble Tea lifecycle and restores terminal state before it
// returns, including cancellation and interrupt paths.
func Run(ctx context.Context, config Config) error {
	if config.Input == nil || config.Output == nil || config.ErrorOutput == nil {
		return errors.New("terminal input, output, and error output are required")
	}

	program := tea.NewProgram(
		newModel(),
		tea.WithContext(ctx),
		tea.WithInput(config.Input),
		tea.WithOutput(config.Output),
	)

	_, err := program.Run()
	if err == nil || errors.Is(err, tea.ErrInterrupted) || errors.Is(err, context.Canceled) {
		return nil
	}

	return fmt.Errorf("run terminal interface: %w", err)
}
