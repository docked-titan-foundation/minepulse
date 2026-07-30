package cmd

import (
	"fmt"
	"io"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
	"github.com/docked-titan-foundation/minepulse/internal/render"
)

// renderOnce writes a single snapshot in the non-interactive output modes.
// (The interactive TUI has its own run loop in watch.go.)
func renderOnce(w io.Writer, out config.Output, s *model.Snapshot) error {
	switch out {
	case config.OutputJSON:
		return render.JSON(w, s)
	default:
		render.Stream(w, s)
		return nil
	}
}

// validateOutput rejects an unknown --output value.
func validateOutput(out config.Output) error {
	switch out {
	case config.OutputTUI, config.OutputStream, config.OutputJSON:
		return nil
	default:
		return fmt.Errorf("invalid --output %q (want tui|stream|json)", out)
	}
}
