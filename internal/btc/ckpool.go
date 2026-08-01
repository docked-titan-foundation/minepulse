package btc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// ckpool publishes its status as JSON records marked with these prefixes. It
// writes them to its logfile at NOTICE level, so they only reach a pod's log
// stream when the operator routes that file there (research R1a) — the parser is
// therefore opportunistic, and "no records" is a normal outcome, not an error.
const (
	poolMarker = "Pool:"
	userMarker = "User "
)

// ckPoolCounters is ckpool's first status record.
type ckPoolCounters struct {
	Runtime      *int64 `json:"runtime"`
	LastUpdate   *int64 `json:"lastupdate"`
	Users        *int   `json:"Users"`
	Workers      *int   `json:"Workers"`
	Idle         *int   `json:"Idle"`
	Disconnected *int   `json:"Disconnected"`
}

// ckPoolHashrates is ckpool's second status record; every value is a
// magnitude-suffixed string (see ParseSuffixed).
type ckPoolHashrates struct {
	H1m  *string `json:"hashrate1m"`
	H5m  *string `json:"hashrate5m"`
	H15m *string `json:"hashrate15m"`
	H1hr *string `json:"hashrate1hr"`
	H1d  *string `json:"hashrate1d"`
}

// ckPoolShares is ckpool's third status record.
type ckPoolShares struct {
	Diff      *float64 `json:"diff"`
	Accepted  *float64 `json:"accepted"`
	Rejected  *float64 `json:"rejected"`
	BestShare *float64 `json:"bestshare"`
	SPS1m     *float64 `json:"SPS1m"`
}

// ckUser is one payout address's record.
type ckUser struct {
	H1m       *string  `json:"hashrate1m"`
	LastShare *int64   `json:"lastshare"`
	Workers   *int     `json:"workers"`
	Shares    *float64 `json:"shares"`
	BestShare *float64 `json:"bestshare"`
	BestEver  *float64 `json:"bestever"`
	// ckpool spells the key "authorised"; the tag matches its wire format.
	Authorized *int64 `json:"authorised"`
}

// ParseStatusLog reads ckpool's periodic status records out of a pod log and
// returns the pool aggregate plus one row per payout address.
//
// The most recent occurrence of each record wins, and the three Pool: records
// are told apart by their keys rather than their order. A log with no records
// yields (nil, nil, nil): ckpool may simply not be routing them here, and a zero
// hashrate would be a claim minepulse cannot support.
func ParseStatusLog(logs []byte) (*model.BitcoinStats, []model.BitcoinMiner, error) {
	if len(logs) == 0 {
		return nil, nil, nil
	}

	var (
		counters  *ckPoolCounters
		hashrates *ckPoolHashrates
		shares    *ckPoolShares
		users     = map[string]ckUser{}
		order     []string
	)

	sc := bufio.NewScanner(bytes.NewReader(logs))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.Contains(line, poolMarker):
			body := jsonSuffix(line)
			if body == "" {
				continue
			}
			// Identify the record by which keys decode into it.
			var c ckPoolCounters
			if json.Unmarshal([]byte(body), &c) == nil && c.LastUpdate != nil {
				counters = &c
				continue
			}
			var h ckPoolHashrates
			if json.Unmarshal([]byte(body), &h) == nil && h.H1m != nil {
				hashrates = &h
				continue
			}
			var s ckPoolShares
			if json.Unmarshal([]byte(body), &s) == nil && s.BestShare != nil {
				shares = &s
			}
		case strings.Contains(line, userMarker):
			name, body := userRecord(line)
			if name == "" || body == "" {
				continue
			}
			var u ckUser
			if json.Unmarshal([]byte(body), &u) != nil {
				continue
			}
			if _, seen := users[name]; !seen {
				order = append(order, name)
			}
			users[name] = u
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	if counters == nil && hashrates == nil && shares == nil {
		return nil, nil, nil
	}

	stats := model.NewBitcoinStats()
	stats.HashrateWindow = "1m"
	if c := counters; c != nil {
		setInt(&stats.Users, c.Users)
		setInt(&stats.Workers, c.Workers)
		setInt(&stats.WorkersIdle, c.Idle)
		setInt(&stats.Disconnected, c.Disconnected)
		if c.Runtime != nil {
			stats.Runtime = time.Duration(*c.Runtime) * time.Second
		}
		stats.LastUpdate = unixTime(c.LastUpdate)
	}
	if h := hashrates; h != nil {
		stats.Hashrate1m = suffixed(h.H1m)
		stats.Hashrate5m = suffixed(h.H5m)
		stats.Hashrate1h = suffixed(h.H1hr)
		stats.Hashrate1d = suffixed(h.H1d)
	}
	if s := shares; s != nil {
		setFloat(&stats.Accepted, s.Accepted)
		setFloat(&stats.Rejected, s.Rejected)
		setFloat(&stats.BestShare, s.BestShare)
		setFloat(&stats.NetworkDiffPct, s.Diff)
		setFloat(&stats.SPS1m, s.SPS1m)
	}

	miners := make([]model.BitcoinMiner, 0, len(order))
	for _, name := range order {
		u := users[name]
		m := model.BitcoinMiner{
			Name:           name,
			Hashrate:       suffixed(u.H1m),
			BestDifficulty: model.Unknown,
			BestEver:       model.Unknown,
			Shares:         model.Unknown,
			Workers:        -1,
			LastSeen:       unixTime(u.LastShare),
			StartTime:      unixTime(u.Authorized),
		}
		setFloat(&m.BestDifficulty, u.BestShare)
		setFloat(&m.BestEver, u.BestEver)
		setFloat(&m.Shares, u.Shares)
		setInt(&m.Workers, u.Workers)
		miners = append(miners, m)
	}
	return stats, miners, nil
}

// jsonSuffix returns the JSON object at the end of a log line. The prefix — a
// ckpool timestamp, a container runtime's own framing, or nothing — is not part
// of any contract, so it is simply skipped.
func jsonSuffix(line string) string {
	i := strings.Index(line, "{")
	if i < 0 || !strings.HasSuffix(strings.TrimSpace(line), "}") {
		return ""
	}
	return strings.TrimSpace(line[i:])
}

// userRecord splits "User <name>:{…}" into its name and JSON body.
func userRecord(line string) (name, body string) {
	i := strings.Index(line, userMarker)
	if i < 0 {
		return "", ""
	}
	rest := line[i+len(userMarker):]
	j := strings.Index(rest, ":{")
	if j < 0 {
		return "", ""
	}
	name = strings.TrimSpace(rest[:j])
	body = jsonSuffix(rest[j:])
	if name == "" || body == "" {
		return "", ""
	}
	return name, body
}

func suffixed(s *string) float64 {
	if s == nil {
		return model.Unknown
	}
	return ParseSuffixed(*s)
}

func setFloat(dst *float64, v *float64) {
	if v != nil {
		*dst = *v
	}
}

func setInt(dst *int, v *int) {
	if v != nil {
		*dst = *v
	}
}

func unixTime(sec *int64) time.Time {
	if sec == nil || *sec <= 0 {
		return time.Time{}
	}
	return time.Unix(*sec, 0)
}
