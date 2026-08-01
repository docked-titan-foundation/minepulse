package collect

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// mockSource renders the full experience with synthetic data and no cluster
// (Constitution III). It simulates a small fleet including one donate-pool
// fallback and one crash-looping node, with CPU that random-walks so the
// sparklines move.
type mockSource struct {
	rng       *rand.Rand
	nodes     []*mockNode
	wallet    string
	noPool    bool
	noBTC     bool
	poolPaid  float64
	poolDue   float64
	btcShares float64
	tick      int
}

type mockNode struct {
	name     string
	pod      string
	phase    string
	restarts int32
	start    time.Time
	cores    int64 // logical CPUs
	capMilli int64
	hashPerT float64 // H/s per thread
	threads  int
	donate   bool
	crash    bool
	good     int64
	total    int64
	// simulated node load excluding the miner, as a fraction 0..1
	otherLoad float64
	hist      *model.Ring
}

// NewMock returns a Source driven entirely by synthetic data.
func NewMock(cfg config.Config) Source {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()
	mk := func(name string, cores int64, donate, crash bool) *mockNode {
		return &mockNode{
			name: name, pod: "monero-idle-miner-" + name[:3] + randSuffix(rng),
			phase: "Running", start: now.Add(-time.Duration(1+rng.Intn(48)) * time.Hour),
			cores: cores, capMilli: cores * 1000, hashPerT: 55 + rng.Float64()*25,
			threads: int(cores) * 3 / 4, donate: donate, crash: crash,
			otherLoad: 0.1 + rng.Float64()*0.2,
			hist:      model.NewRing(120),
		}
	}
	m := &mockSource{
		rng:      rng,
		wallet:   "47mbhuNjTTUWF2zSYU4DZVRSwC7pvfjjM8FTtzzku41rBQxxdhHh8kA6gnmRyCQj7X8SgeGZL8HiRfiHN6uUhmtaEA5d8pM",
		noPool:   cfg.NoPool,
		noBTC:    cfg.NoBTC,
		poolPaid: 0.184,
		poolDue:  0.0031,
	}
	if cfg.Wallet != "" {
		m.wallet = cfg.Wallet
	}
	m.nodes = []*mockNode{
		mk("andromeda", 8, false, false),
		mk("cygnus", 4, false, false),
		mk("orion", 6, true, false), // donate-pool fallback
		mk("draco", 4, false, true), // crashlooping
	}
	m.nodes[3].phase = "CrashLoopBackOff"
	m.nodes[3].restarts = 7
	return m
}

func randSuffix(rng *rand.Rand) string {
	const alpha = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 5)
	for i := range b {
		b[i] = alpha[rng.Intn(len(alpha))]
	}
	return string(b)
}

func (m *mockSource) Close() error { return nil }

func (m *mockSource) Gather(_ context.Context) (*model.Snapshot, error) {
	m.tick++
	now := time.Now()
	snap := &model.Snapshot{Timestamp: now}

	for _, n := range m.nodes {
		ns := model.NodeStatus{
			Node: n.name, Pod: n.pod, Phase: n.phase, Restarts: n.restarts,
			Image:  "metal3d/xmrig:6.20.0-2",
			Uptime: now.Sub(n.start), History: n.hist, StatsSource: model.SourceAPI,
		}

		if n.crash {
			ns.StatsSource = model.SourceNone
			ns.Note = "pod not Running"
			ns.Uptime = 0
			snap.Nodes = append(snap.Nodes, ns)
			continue
		}

		// Random-walk the "other" (non-miner) load; the miner takes the rest.
		n.otherLoad += (m.rng.Float64() - 0.5) * 0.15
		n.otherLoad = math.Max(0, math.Min(0.85, n.otherLoad))
		minerFrac := math.Max(0, 0.9-n.otherLoad) // miner backs off as others rise
		minerMilli := int64(minerFrac * float64(n.capMilli))
		usedMilli := int64((n.otherLoad)*float64(n.capMilli)) + minerMilli
		if usedMilli > n.capMilli {
			usedMilli = n.capMilli
		}
		cpu := &model.CPUSample{
			T: now, MinerMilli: minerMilli, NodeUsedMilli: usedMilli,
			NodeCapacityMilli: n.capMilli,
			FreePct:           float64(n.capMilli-usedMilli) / float64(n.capMilli) * 100,
		}
		n.hist.Append(*cpu)
		ns.CPU = cpu

		// Hashrate tracks the miner's share of CPU. The miner targets ~75% of
		// threads (matching cpuFreeThresholdPercent=25); scale actual active
		// threads by how much of that CPU it currently gets.
		minerPct := float64(minerMilli) / float64(n.capMilli) * 100
		activeThreads := int(math.Round(float64(n.threads) * minerPct / 75))
		if activeThreads > n.threads {
			activeThreads = n.threads
		}
		hr := float64(activeThreads) * n.hashPerT
		n.good += int64(m.rng.Intn(3))
		n.total = n.good + int64(m.tick/7)
		pool := "pool.supportxmr.com:443"
		if n.donate {
			pool = "donate.v2.xmrig.com:3333"
		}
		ns.Mining = &model.MiningStats{
			Hashrate10s: hr * (0.9 + m.rng.Float64()*0.2), Hashrate60s: hr, Hashrate15m: hr * 0.98,
			ThreadsActive: activeThreads, ThreadsTotal: int(n.cores),
			SharesGood: n.good, SharesTotal: n.total,
			Pool: pool, Connected: true, PingMs: 20 + m.rng.Intn(60),
			DonateFallback: n.donate, WorkerID: n.name, Version: "6.20.0",
		}
		snap.Nodes = append(snap.Nodes, ns)
	}

	if !m.noPool {
		m.poolDue += 0.00002
		var reported float64
		for _, n := range m.nodes {
			if !n.crash && !n.donate {
				reported += float64(n.threads) * n.hashPerT
			}
		}
		snap.Pool = &model.PoolStats{
			Wallet: m.wallet, ReportedHashrate: reported,
			AmountDueXMR: m.poolDue, AmountPaidXMR: m.poolPaid,
			TotalHashes: int64(4_200_000_000 + m.tick*1_000_000),
			LastShare:   now.Add(-time.Duration(m.rng.Intn(90)) * time.Second),
			AsOf:        now,
		}
	}

	if !m.noBTC {
		snap.Bitcoin = m.bitcoin(now)
	}

	if m.tick == 1 {
		snap.Warnings = append(snap.Warnings, "mock mode: synthetic data, no cluster")
	}
	snap.Summarize()
	return snap, nil
}

// bitcoin fabricates both halves of the Bitcoin tab: a public-pool that reports
// per-device rows over its API, and a ckpool whose stats come from log records —
// so every branch of the renderer is exercised without a cluster (FR-014).
func (m *mockSource) bitcoin(now time.Time) *model.BitcoinView {
	m.btcShares += float64(m.rng.Intn(3))
	jitter := func(base float64) float64 { return base * (0.97 + m.rng.Float64()*0.06) }

	ppStats := model.NewBitcoinStats()
	ppStats.HashrateWindow = "now"
	ppStats.Hashrate1m = jitter(1.68e12)
	ppStats.Workers = 2
	ppStats.BestShare = 4.1e9
	ppStats.BlockHeight = 907213 + int64(m.tick/20)
	ppStats.BlocksFound = 0
	ppStats.TotalsAsOf = now.Add(-2 * time.Minute)
	ppStats.LastUpdate = now

	ckStats := model.NewBitcoinStats()
	ckStats.HashrateWindow = "1m"
	ckStats.Hashrate1m = jitter(480e12)
	ckStats.Hashrate5m = jitter(472e12)
	ckStats.Hashrate1h = jitter(468e12)
	ckStats.Users = 1
	ckStats.Workers = 2
	ckStats.WorkersIdle = 0
	ckStats.Disconnected = 0
	ckStats.Accepted = 12470 + m.btcShares
	ckStats.Rejected = 3
	ckStats.BestShare = 1.2e9
	ckStats.NetworkDiffPct = 0.02
	ckStats.SPS1m = 1.4
	ckStats.LastUpdate = now.Add(-8 * time.Second)
	ckStats.Runtime = 512000 * time.Second

	return &model.BitcoinView{
		Scope: "all namespaces (mock)",
		Pools: []model.BitcoinPool{
			{
				Impl: model.ImplPublicPool, Namespace: "bitcoin", Pod: "mining-pool-6f9c4b7d5-2k9xp",
				Node: "andromeda", Phase: "Running", Running: true, Uptime: 73 * time.Hour,
				Source: model.SourceAPI, Detail: model.DetailDevice, Stats: ppStats, AsOf: now,
				Miners: []model.BitcoinMiner{
					{
						Name: "bitaxe-01", SessionID: "a1b2c3d4", Hashrate: jitter(480e9),
						BestDifficulty: 1.2e9, BestEver: model.Unknown, Shares: model.Unknown, Workers: -1,
						StartTime: now.Add(-146 * time.Hour), LastSeen: now.Add(-11 * time.Second),
					},
					{
						Name: "nerdqaxe-01", SessionID: "e5f6a7b8", Hashrate: jitter(1.2e12),
						BestDifficulty: 8.9e8, BestEver: model.Unknown, Shares: model.Unknown, Workers: -1,
						StartTime: now.Add(-64 * time.Hour), LastSeen: now.Add(-13 * time.Second),
					},
				},
			},
			{
				Impl: model.ImplCkpool, Namespace: "bitcoin-solo", Pod: "mining-pool-7c9f8b6d4-x2k9p",
				Node: "orion", Phase: "Running", Running: true, Uptime: 142 * time.Hour,
				Source: model.SourceLogs, Detail: model.DetailAddress, Stats: ckStats, AsOf: now,
				Note: "per-device detail unavailable from ckpool logs (addresses only)",
				Miners: []model.BitcoinMiner{
					{
						Name:     "bc1qexampleaddressxxxxxxxxxxxxxxxxxxxxxxx",
						Hashrate: ckStats.Hashrate1m, BestDifficulty: 1.2e9, BestEver: 4.1e9,
						Workers: 2, Shares: ckStats.Accepted,
						StartTime: now.Add(-142 * time.Hour), LastSeen: now.Add(-time.Duration(m.rng.Intn(40)) * time.Second),
					},
				},
			},
		},
	}
}
