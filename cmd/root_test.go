package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// The environment fills only what the command line left alone, so a value that
// lives in a shell profile never overrides one typed deliberately.
func TestApplyEnv(t *testing.T) {
	const addr = "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx"

	tests := []struct {
		name     string
		env      map[string]string
		args     []string
		wantAddr string
		wantTab  string
	}{
		{
			name:     "environment fills an unset flag",
			env:      map[string]string{"MINEPULSE_BTC_ADDRESS": addr, "MINEPULSE_TAB": "bitcoin"},
			wantAddr: addr, wantTab: "bitcoin",
		},
		{
			name:     "an explicit flag beats the environment",
			env:      map[string]string{"MINEPULSE_TAB": "bitcoin"},
			args:     []string{"--tab", "monero"},
			wantAddr: "", wantTab: "monero",
		},
		{
			name:     "an empty variable is not a value",
			env:      map[string]string{"MINEPULSE_BTC_ADDRESS": ""},
			wantAddr: "", wantTab: "monero",
		},
		{
			name:     "unrelated variables are ignored",
			env:      map[string]string{"BTC_ADDRESS": addr},
			wantAddr: "", wantTab: "monero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			var address, tab string
			cmd := &cobra.Command{Use: "test", Run: func(*cobra.Command, []string) {}}
			cmd.Flags().StringVar(&address, "btc-address", "", "")
			cmd.Flags().StringVar(&tab, "tab", "monero", "")
			if err := cmd.Flags().Parse(tt.args); err != nil {
				t.Fatalf("parse: %v", err)
			}

			applyEnv(cmd)

			if address != tt.wantAddr {
				t.Errorf("btc-address = %q, want %q", address, tt.wantAddr)
			}
			if tab != tt.wantTab {
				t.Errorf("tab = %q, want %q", tab, tt.wantTab)
			}
		})
	}
}

// A dashed flag maps to an underscored variable, which is the rule the help
// text promises.
func TestApplyEnvNameMapping(t *testing.T) {
	t.Setenv("MINEPULSE_NO_BTC", "true")
	var noBTC bool
	cmd := &cobra.Command{Use: "test", Run: func(*cobra.Command, []string) {}}
	cmd.Flags().BoolVar(&noBTC, "no-btc", false, "")
	if err := cmd.Flags().Parse(nil); err != nil {
		t.Fatalf("parse: %v", err)
	}

	applyEnv(cmd)

	if !noBTC {
		t.Error("MINEPULSE_NO_BTC did not reach --no-btc")
	}
}
