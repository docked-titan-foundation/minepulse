package poolid

import (
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// Brand may fall back to the domain; mining mode may not fall back to anything.
// Solo and shared decide whether an operator ever sees a payout, and a hostname
// cannot settle it — so an unrecognized pool gets a name and no mode (FR-005/6).
func TestIdentify(t *testing.T) {
	for _, tc := range []struct {
		addr  string
		brand string
		mode  model.MiningMode
	}{
		{"pool.supportxmr.com:443", "SupportXMR", model.ModeShared},
		{"gulf.moneroocean.stream:10128", "MoneroOcean", model.ModeShared},
		{"donate.v2.xmrig.com:3333", "XMRig donate", model.ModeShared},
		{"monero.hashvault.pro:443", "HashVault", model.ModeShared},
		// Unknown pool: the registrable domain is, in practice, its name.
		{"stratum.some-new-pool.io:4444", "some-new-pool.io", model.ModeUnknown},
		{"a.b.c.example.co.uk:3333", "example.co.uk", model.ModeUnknown},
		// A bare address has no name to give, and must not have one invented.
		{"10.43.7.12:3333", "", model.ModeUnknown},
		{"116.202.180.221:443", "", model.ModeUnknown},
		{"[2001:db8::1]:3333", "", model.ModeUnknown},
		// Cluster-local names: the Bitcoin collector supplies both fields for
		// these directly, and the DNS name says nothing about a brand.
		{"mining-pool.bitcoin.svc:3333", "", model.ModeUnknown},
		{"mining-pool.bitcoin.svc.cluster.local:3333", "", model.ModeUnknown},
		{"localhost:3333", "", model.ModeUnknown},
		{"", "", model.ModeUnknown},
	} {
		brand, mode := Identify(tc.addr)
		if brand != tc.brand || mode != tc.mode {
			t.Errorf("Identify(%q) = (%q, %q), want (%q, %q)",
				tc.addr, brand, mode, tc.brand, tc.mode)
		}
	}
}

func TestHost(t *testing.T) {
	for addr, want := range map[string]string{
		"pool.supportxmr.com:443": "pool.supportxmr.com",
		"pool.supportxmr.com":     "pool.supportxmr.com",
		"[2001:db8::1]:3333":      "2001:db8::1",
		"":                        "",
	} {
		if got := Host(addr); got != want {
			t.Errorf("Host(%q) = %q, want %q", addr, got, want)
		}
	}
}
