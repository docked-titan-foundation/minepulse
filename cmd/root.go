// Package cmd is minepulse's cobra command surface.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/docked-titan-foundation/minepulse/internal/config"
)

var cfg = config.Defaults()

var rootCmd = &cobra.Command{
	Use:   "minepulse",
	Short: "Live view of Monero mining across a Kubernetes cluster",
	Long: `minepulse is a read-only terminal scope for the monero-idle-miner DaemonSet.

It shows, continuously, how mining is going across the cluster: per-node hashrate,
threads, shares and pool status; the miner's CPU versus each node's free CPU over
time; and pool-side earnings for your wallet. It never changes anything it observes.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		// Resolve the default output from whether stdout is a terminal, unless
		// the user set --output explicitly.
		if !cmd.Flags().Changed("output") {
			if isTerminal(os.Stdout) {
				cfg.Output = config.OutputTUI
			} else {
				cfg.Output = config.OutputStream
			}
		}
	},
}

// Execute runs the root command with the given build version.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "minepulse: "+err.Error())
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&cfg.Namespace, "namespace", "n", cfg.Namespace, "namespace the miner runs in")
	pf.StringVar(&cfg.Selector, "selector", cfg.Selector, "label selector for miner pods")
	pf.DurationVar(&cfg.Interval, "interval", cfg.Interval, "refresh interval")
	pf.StringVarP((*string)(&cfg.Output), "output", "o", string(cfg.Output), "output: tui|stream|json")
	pf.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "path to kubeconfig (default: env/well-known)")
	pf.StringVar(&cfg.Context, "context", "", "kubeconfig context to use")
	pf.StringVar(&cfg.Wallet, "wallet", "", "XMR address for the pool panel (default: auto-detect from the miner)")
	pf.BoolVar(&cfg.NoPool, "no-pool", false, "do not query the mining pool")
	pf.StringVar(&cfg.PoolAPIURL, "pool-api", cfg.PoolAPIURL, "pool stats API base URL (SupportXMR-compatible)")
	pf.StringVar((*string)(&cfg.XMRigAPI), "xmrig-api", string(cfg.XMRigAPI), "miner stats source: auto|on|off")
	pf.BoolVar(&cfg.Mock, "mock", false, "use synthetic data; no cluster access")

	rootCmd.AddCommand(watchCmd, snapshotCmd, doctorCmd)
}

// isTerminal reports whether f is a character device (a TTY).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
