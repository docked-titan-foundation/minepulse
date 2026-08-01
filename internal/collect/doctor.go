package collect

import (
	"context"
	"fmt"
	"time"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/diag"
	"github.com/docked-titan-foundation/minepulse/internal/pool"
)

// RunDoctor performs a single, bounded, read-only preflight pass and returns a
// report. A hard prerequisite failure (bad kubeconfig, unreachable cluster) is
// recorded as a failing check and stops dependent checks cleanly rather than
// erroring the whole command (Constitution III). It never mutates anything
// (Constitution II).
func RunDoctor(ctx context.Context, cfg config.Config) (*diag.Report, error) {
	rep := &diag.Report{}

	k, err := newKubeClient(cfg)
	if err != nil {
		rep.Add(diag.Check{Name: "kubeconfig", Status: diag.StatusFail, Detail: err.Error(),
			Remedy: "Check --kubeconfig/--context and that the file is valid."})
		return rep, nil
	}

	miners, err := k.listMiners(ctx)
	if err != nil {
		rep.Add(diag.Check{Name: "cluster API", Status: diag.StatusFail, Detail: err.Error(),
			Remedy: "Verify cluster connectivity and read access (see deploy/rbac.yaml)."})
		return rep, nil
	}
	rep.Add(diag.Check{Name: "cluster API", Status: diag.StatusOK, Detail: "reachable"})

	running := 0
	for i := range miners {
		if miners[i].running {
			running++
		}
	}
	if len(miners) == 0 {
		rep.Add(diag.Check{Name: "miner pods", Status: diag.StatusWarn,
			Detail: fmt.Sprintf("no pods match %q in namespace %q", cfg.Selector, cfg.Namespace),
			Remedy: "Check -n/--namespace and --selector; is monero-idle-miner installed?"})
	} else {
		rep.Add(diag.Check{Name: "miner pods", Status: diag.StatusOK,
			Detail: fmt.Sprintf("%d found, %d Running", len(miners), running)})
	}

	if _, e := k.nodeCPUUsed(ctx); e != nil {
		rep.Add(diag.Check{Name: "CPU metrics", Status: diag.StatusWarn, Detail: "metrics API unavailable",
			Remedy: "Install metrics-server; CPU columns read n/a without it."})
	} else {
		rep.Add(diag.Check{Name: "CPU metrics", Status: diag.StatusOK, Detail: "metrics-server responding"})
	}

	// Headline check: probe the XMRig HTTP API on each running miner.
	reachable := 0
	for i := range miners {
		if !miners[i].running {
			continue
		}
		if _, e := k.proxyGet(ctx, miners[i].pod, "1/summary"); e == nil {
			reachable++
		}
	}
	rep.Add(diag.APIReachabilityCheck(running, reachable))

	rep.Add(poolCheck(ctx, cfg, miners))
	rep.Add(bitcoinCheck(ctx, cfg, k))
	return rep, nil
}

// bitcoinCheck reports what Bitcoin pool discovery found and, when a pool is
// there but silent, what the operator can do about it (FR-018). A missing pool
// is INFO — plenty of clusters mine only Monero — so it never fails the exit
// code on its own.
func bitcoinCheck(ctx context.Context, cfg config.Config, k *kubeClient) diag.Check {
	if cfg.NoBTC {
		return diag.Check{Name: "bitcoin pool", Status: diag.StatusInfo, Detail: "skipped (--no-btc)"}
	}

	pods, scope, warn := k.detectPools(ctx, cfg)
	if warn != "" || len(pods) == 0 {
		return diag.BitcoinCheck(scope, nil, false, warn)
	}

	c := newBTCCollector(k, cfg)
	results := make([]diag.BitcoinPoolResult, 0, len(pods))
	for i := range pods {
		p, _ := c.gatherPool(ctx, &pods[i], time.Now())
		results = append(results, diag.BitcoinPoolResult{
			Where:    fmt.Sprintf("%s in %s/%s", p.Impl, p.Namespace, p.Pod),
			Source:   string(p.Source),
			HasStats: p.Stats != nil,
			Remedy:   p.Remedy,
		})
	}
	return diag.BitcoinCheck(scope, results, false, "")
}

func poolCheck(ctx context.Context, cfg config.Config, miners []minerPod) diag.Check {
	if cfg.NoPool {
		return diag.Check{Name: "pool API", Status: diag.StatusInfo, Detail: "skipped (--no-pool)"}
	}
	wallet := cfg.Wallet
	if wallet == "" {
		for i := range miners {
			if miners[i].wallet != "" {
				wallet = miners[i].wallet
				break
			}
		}
	}
	if wallet == "" {
		return diag.Check{Name: "pool API", Status: diag.StatusInfo,
			Detail: "no wallet resolved from the miner", Remedy: "Pass --wallet to enable the pool panel."}
	}
	if _, e := pool.NewClient(cfg.PoolAPIURL).MinerStats(ctx, wallet); e != nil {
		return diag.Check{Name: "pool API", Status: diag.StatusWarn, Detail: "unreachable: " + e.Error(),
			Remedy: "Check egress to " + cfg.PoolAPIURL + "; the earnings panel will show stale."}
	}
	return diag.Check{Name: "pool API", Status: diag.StatusOK, Detail: "reachable"}
}
