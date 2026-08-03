// Package poolid names a mining pool from its address: which brand runs there,
// and whether it shares work with other miners or hands out solo jobs. Both are
// pure functions of a string so they can be table-tested with no cluster and no
// network (Constitution IV).
package poolid

import (
	"net"
	"strings"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// known maps a registrable domain to the brand that runs it and how that pool
// shares work. Only pools whose model is a matter of public record belong here —
// a guess in this table becomes a confident wrong answer on screen.
var known = map[string]struct {
	brand string
	mode  model.MiningMode
}{
	"supportxmr.com":     {"SupportXMR", model.ModeShared},
	"moneroocean.stream": {"MoneroOcean", model.ModeShared},
	"hashvault.pro":      {"HashVault", model.ModeShared},
	"nanopool.org":       {"Nanopool", model.ModeShared},
	"2miners.com":        {"2Miners", model.ModeShared},
	"c3pool.com":         {"C3Pool", model.ModeShared},
	"xmrpool.eu":         {"XMRPool.eu", model.ModeShared},
	// XMRig's built-in donation pool. Named so the identity line says out loud
	// where a fallen-back miner's work is going.
	"xmrig.com": {"XMRig donate", model.ModeShared},
}

// Identify names the pool at hostPort and says how it shares work.
//
// An address nobody recognizes still gets a brand — its registrable domain,
// which in practice *is* the pool's name — but never a mining mode: solo and
// shared are not guessable from a hostname, and the difference decides whether
// the operator ever sees a payout. A bare IP has no name to give.
func Identify(hostPort string) (brand string, mode model.MiningMode) {
	host := Host(hostPort)
	if host == "" {
		return "", model.ModeUnknown
	}
	if net.ParseIP(host) != nil {
		return "", model.ModeUnknown
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))

	// A cluster-local name (mining-pool.bitcoin.svc.cluster.local) is a
	// self-hosted pool: the domain says nothing about a brand, and the Bitcoin
	// collector supplies both fields directly for those.
	if strings.HasSuffix(host, ".svc") || strings.Contains(host, ".svc.") ||
		strings.HasSuffix(host, ".local") || !strings.Contains(host, ".") {
		return "", model.ModeUnknown
	}

	dom := registrable(host)
	if k, ok := known[dom]; ok {
		return k.brand, k.mode
	}
	return dom, model.ModeUnknown
}

// Host strips the port from a pool address, tolerating IPv6 literals.
func Host(hostPort string) string {
	if hostPort == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(hostPort); err == nil {
		return h
	}
	return strings.Trim(hostPort, "[]")
}

// registrable reduces a hostname to the domain someone registered:
// pool.supportxmr.com → supportxmr.com. It handles the two-label public suffixes
// a mining pool is plausibly on (co.uk and friends) and otherwise takes the last
// two labels, which is right for every pool in the table above.
func registrable(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return host
	}
	last2 := strings.Join(parts[len(parts)-2:], ".")
	if len(parts) >= 3 && twoLabelSuffix[last2] {
		return strings.Join(parts[len(parts)-3:], ".")
	}
	return last2
}

var twoLabelSuffix = map[string]bool{
	"co.uk": true, "org.uk": true, "com.au": true, "co.nz": true,
	"com.br": true, "co.za": true, "com.mx": true, "co.jp": true,
}
