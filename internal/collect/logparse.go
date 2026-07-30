package collect

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/docked-titan-foundation/minepulse/internal/model"
	"github.com/docked-titan-foundation/minepulse/internal/xmrig"
)

var (
	reANSI     = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	reSpeed    = regexp.MustCompile(`speed\s+10s/60s/15m\s+(\S+)\s+(\S+)\s+(\S+)\s+H/s`)
	reAccepted = regexp.MustCompile(`accepted\s+\((\d+)/(\d+)\)`)
	reReady    = regexp.MustCompile(`READY threads\s+(\d+)/(\d+)`)
	reUsePool  = regexp.MustCompile(`use pool\s+(\S+)`)
	rePoolBan  = regexp.MustCompile(`POOL #1\s+(\S+)`)
	reVersion  = regexp.MustCompile(`XMRig/(\S+)`)
)

// ParseLogs derives best-effort mining stats from an XMRig pod's log output.
// It is the reduced-fidelity fallback used when the XMRig HTTP API is off
// (Constitution III). Returns nil if the logs contain no minable signal.
// configuredPool enables donate-fallback detection.
func ParseLogs(logs, configuredPool string) *model.MiningStats {
	logs = reANSI.ReplaceAllString(logs, "")

	ms := model.MiningStats{
		Hashrate10s: model.Unknown, Hashrate60s: model.Unknown, Hashrate15m: model.Unknown,
		ThreadsActive: -1, ThreadsTotal: -1, PingMs: -1,
	}
	found := false

	// Scan every line; keep the most recent value for each signal.
	for _, line := range strings.Split(logs, "\n") {
		if m := reSpeed.FindStringSubmatch(line); m != nil {
			ms.Hashrate10s = parseHR(m[1])
			ms.Hashrate60s = parseHR(m[2])
			ms.Hashrate15m = parseHR(m[3])
			found = true
		}
		if m := reAccepted.FindStringSubmatch(line); m != nil {
			ms.SharesGood = atoi64(m[1])
			ms.SharesTotal = atoi64(m[2])
			found = true
		}
		if m := reReady.FindStringSubmatch(line); m != nil {
			ms.ThreadsActive = atoi(m[1])
			ms.ThreadsTotal = atoi(m[2])
			found = true
		}
		if m := reUsePool.FindStringSubmatch(line); m != nil {
			ms.Pool = m[1]
			ms.Connected = true
			found = true
		} else if m := rePoolBan.FindStringSubmatch(line); m != nil && ms.Pool == "" {
			ms.Pool = m[1]
			found = true
		}
		if m := reVersion.FindStringSubmatch(line); m != nil && ms.Version == "" {
			ms.Version = m[1]
		}
	}

	if !found {
		return nil
	}
	if ms.Pool != "" {
		ms.DonateFallback = xmrig.IsDonatePool(ms.Pool, configuredPool)
	}
	return &ms
}

func parseHR(s string) float64 {
	if s == "n/a" || s == "-" {
		return model.Unknown
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return model.Unknown
	}
	return v
}

func atoi(s string) int     { v, _ := strconv.Atoi(s); return v }
func atoi64(s string) int64 { v, _ := strconv.ParseInt(s, 10, 64); return v }
