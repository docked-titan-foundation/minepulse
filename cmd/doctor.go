package cmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/docked-titan-foundation/minepulse/internal/collect"
	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/render"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Preflight the cluster: can minepulse read the miners, and is the XMRig API enabled?",
	Long: `doctor runs a one-pass, read-only preflight and prints a checklist.

Its headline check is whether the XMRig HTTP API is reachable on the miners — if it
is not, minepulse falls back to log parsing (reduced fidelity), so doctor warns and
recommends enabling httpApi.enabled: true on the monero-idle-miner chart. It also
checks cluster reachability, miner discovery, CPU metrics, and the pool API.

Exits non-zero only when a hard prerequisite fails; warnings do not.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := validateOutput(cfg.Output); err != nil {
			return err
		}
		rep, err := collect.RunDoctor(context.Background(), cfg)
		if err != nil {
			return err
		}
		if cfg.Output == config.OutputJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(rep); err != nil {
				return err
			}
		} else {
			render.Doctor(os.Stdout, rep)
		}
		if rep.Failed() {
			os.Exit(1)
		}
		return nil
	},
}
