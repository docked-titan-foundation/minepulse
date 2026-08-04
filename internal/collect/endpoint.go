package collect

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/docked-titan-foundation/minepulse/internal/model"
	"github.com/docked-titan-foundation/minepulse/internal/poolid"
)

// moneroEndpoint is the pool the Monero tab's identity line describes: the one
// most miners are actually connected to, identified and located.
//
// "Most" matters. When a node has silently fallen back to XMRig's donate pool
// the miners no longer agree, and a summary line that quietly reported the
// majority as the whole truth would hide the single thing this tool exists to
// catch. The majority pool is named and the disagreement is flagged (FR-007).
func moneroEndpoint(nodes []model.NodeStatus, cn clusterNet, configured string) *model.PoolEndpoint {
	count := map[string]int{}
	ipFor := map[string]string{}
	for i := range nodes {
		m := nodes[i].Mining
		if m == nil || m.Pool == "" {
			continue
		}
		count[m.Pool]++
		if m.PoolIP != "" && ipFor[m.Pool] == "" {
			ipFor[m.Pool] = m.PoolIP
		}
	}

	if len(count) == 0 {
		// Nothing has connected yet. The configured pool is still worth naming;
		// there is simply no resolved address to locate it by.
		if configured == "" {
			return nil
		}
		brand, mode := poolid.Identify(configured)
		return &model.PoolEndpoint{
			URL: configured, Brand: brand, Mode: mode,
			Locality: model.LocalityUnknown, Basis: "no miner has connected yet",
		}
	}

	// Sort for a stable winner: most nodes first, then the configured pool, then
	// by name — so the line does not flicker between equally-popular pools.
	pools := make([]string, 0, len(count))
	for p := range count {
		pools = append(pools, p)
	}
	sort.Slice(pools, func(i, j int) bool {
		if count[pools[i]] != count[pools[j]] {
			return count[pools[i]] > count[pools[j]]
		}
		if (pools[i] == configured) != (pools[j] == configured) {
			return pools[i] == configured
		}
		return pools[i] < pools[j]
	})

	win := pools[0]
	brand, mode := poolid.Identify(win)
	loc, basis := cn.classify(ipFor[win])
	return &model.PoolEndpoint{
		URL: win, IP: ipFor[win], Brand: brand, Mode: mode,
		Locality: loc, Basis: basis, Diverged: len(pools) > 1,
	}
}

// bitcoinEndpoint locates one detected pool: the Service miners reach it through
// when there is one, and the pod itself when there is not.
//
// Brand comes from the fingerprint rather than the address, because a
// self-hosted pool's hostname says nothing about what runs there. The mode is
// only asserted where the implementation settles it: public-pool is solo by
// design, while ckpool ships in both flavors and only its image can say which
// this is (FR-006).
func (k *kubeClient) bitcoinEndpoint(ctx context.Context, p poolPod, cn clusterNet) *model.PoolEndpoint {
	url, ip := k.stratumAddress(ctx, p)
	loc, basis := cn.classify(ip)
	return &model.PoolEndpoint{
		URL: url, IP: ip,
		Brand:    string(p.impl),
		Mode:     implMode(p),
		Locality: loc, Basis: basis,
	}
}

func implMode(p poolPod) model.MiningMode {
	switch p.impl {
	case model.ImplPublicPool:
		return model.ModeSolo
	case model.ImplCkpool:
		// ckpool builds as a shared pool or as ckpool-solo. Only the image name
		// distinguishes them from out here, so anything else stays unknown
		// rather than inheriting this repo's solo-mining assumption.
		if strings.Contains(strings.ToLower(p.image), "solo") {
			return model.ModeSolo
		}
		return model.ModeUnknown
	default:
		return model.ModeUnknown
	}
}

// stratumAddress is where miners connect: the Service fronting this pool if one
// selects it, otherwise the pod's own address.
func (k *kubeClient) stratumAddress(ctx context.Context, p poolPod) (url, ip string) {
	fallback := fmt.Sprintf("%s:%d", p.podIP, p.stratum)
	if p.podIP == "" {
		fallback = fmt.Sprintf("%s.%s:%d", p.pod, p.namespace, p.stratum)
	}

	svcs, err := k.cs.CoreV1().Services(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fallback, p.podIP
	}
	for i := range svcs.Items {
		s := &svcs.Items[i]
		if len(s.Spec.Selector) == 0 || !selects(s.Spec.Selector, p.labels) {
			continue
		}
		port := p.stratum
		for _, sp := range s.Spec.Ports {
			if sp.Name == "stratum" {
				port = int(sp.Port)
			}
		}
		clusterIP := s.Spec.ClusterIP
		if clusterIP == "None" {
			clusterIP = p.podIP
		}
		return fmt.Sprintf("%s.%s.svc:%d", s.Name, s.Namespace, port), clusterIP
	}
	return fallback, p.podIP
}

// selects reports whether every label in a Service selector is present on the
// pod with the same value — the same subset rule Kubernetes itself applies.
func selects(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}
