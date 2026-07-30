package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/docked-titan-foundation/minepulse/internal/collect"
	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/render"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuously render cluster mining status until interrupted",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := validateOutput(cfg.Output); err != nil {
			return err
		}
		src, err := collect.New(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Interactive full-screen dashboard.
		if cfg.Output == config.OutputTUI {
			return render.RunTUI(ctx, cfg.Interval, src.Gather)
		}

		// Non-interactive: emit a snapshot per tick.
		out := cfg.Output
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		gather := func() {
			snap, gerr := src.Gather(ctx)
			if gerr != nil {
				fmt.Fprintln(os.Stderr, "minepulse: "+gerr.Error())
				return
			}
			_ = renderOnce(os.Stdout, out, snap)
		}

		gather() // immediate first paint
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				gather()
			}
		}
	},
}
