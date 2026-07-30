package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/docked-titan-foundation/minepulse/internal/collect"
	"github.com/docked-titan-foundation/minepulse/internal/config"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Gather one snapshot, print it, and exit",
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := validateOutput(cfg.Output); err != nil {
			return err
		}
		src, err := collect.New(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		snap, err := src.Gather(context.Background())
		if err != nil {
			return err
		}

		// A one-shot snapshot has no interactive mode; treat tui as stream.
		out := cfg.Output
		if out == config.OutputTUI {
			out = config.OutputStream
		}
		return renderOnce(os.Stdout, out, snap)
	},
}
