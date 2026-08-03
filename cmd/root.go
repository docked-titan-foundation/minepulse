// Package cmd is minepulse's cobra command surface.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/docked-titan-foundation/minepulse/internal/config"
)

var cfg = config.Defaults()

var rootCmd = &cobra.Command{
	Use:   "minepulse",
	Short: "Live view of Monero mining across a Kubernetes cluster",
	Long: `minepulse is a read-only terminal scope for the monero-idle-miner DaemonSet.

It shows, continuously, how mining is going across the cluster: per-node hashrate,
threads, shares and pool status; the miner's CPU versus each node's free CPU over
time; and pool-side earnings for your wallet. It never changes anything it observes.

Press b for the Bitcoin tab: the solo pool running in the same cluster, found
without configuration.

Every flag can also be set from the environment as MINEPULSE_<FLAG>, with dashes
as underscores — MINEPULSE_BTC_ADDRESS, MINEPULSE_TAB, MINEPULSE_NAMESPACE — so
the values you always pass can live in your shell profile. An explicit flag
always wins over the environment.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		applyEnv(cmd)

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

// envPrefix namespaces minepulse's environment variables.
const envPrefix = "MINEPULSE_"

// applyEnv fills any flag the user did not pass from MINEPULSE_<FLAG>, so
// settings that never change — a BTC payout address, a preferred tab — can live
// in a shell profile instead of every command line. A flag given explicitly
// always wins; an unparseable value is reported rather than silently ignored.
func applyEnv(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}
		key := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		val, ok := os.LookupEnv(key)
		if !ok || val == "" {
			return
		}
		if err := f.Value.Set(val); err != nil {
			fmt.Fprintf(os.Stderr, "minepulse: %s=%q ignored: %v\n", key, val, err)
		}
	})
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
	pf.StringVarP((*string)(&cfg.Tab), "tab", "t", string(cfg.Tab), "coin the dashboard opens on: monero|bitcoin")
	pf.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "path to kubeconfig (default: env/well-known)")
	pf.StringVar(&cfg.Context, "context", "", "kubeconfig context to use")
	pf.StringVar(&cfg.Wallet, "wallet", "", "XMR address for the pool panel (default: auto-detect from the miner)")
	pf.BoolVar(&cfg.NoPool, "no-pool", false, "do not query the mining pool")
	pf.StringVar(&cfg.PoolAPIURL, "pool-api", cfg.PoolAPIURL, "pool stats API base URL (SupportXMR-compatible)")
	pf.StringVar((*string)(&cfg.XMRigAPI), "xmrig-api", string(cfg.XMRigAPI), "miner stats source: auto|on|off")
	pf.BoolVar(&cfg.Mock, "mock", false, "use synthetic data; no cluster access")
	pf.StringVar(&cfg.BTCNamespace, "btc-namespace", "", "namespace of the Bitcoin solo pool (default: search all readable namespaces)")
	pf.StringVar(&cfg.BTCSelector, "btc-selector", "", "extra label selector for Bitcoin pool discovery")
	pf.StringVar(&cfg.BTCAddress, "btc-address", "", "BTC payout address, for per-worker rows on public-pool")
	pf.IntVar(&cfg.BTCAPIPort, "btc-api-port", 0, "public-pool API port (default: the pod's \"api\" port, else 3334)")
	pf.BoolVar(&cfg.NoBTC, "no-btc", false, "do not look for a Bitcoin pool")

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
