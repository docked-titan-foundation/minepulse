package collect

import (
	"context"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/docked-titan-foundation/minepulse/internal/config"
	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// defaultPublicPoolAPIPort is what the bitcoin-stack chart uses when the pod
// declares no port named "api".
const defaultPublicPoolAPIPort = 3334

// chartPartOf is the label the mining-pool chart sets on both implementations.
// It corroborates a fingerprint; it never establishes one on its own, because
// the chart names the workload the same either way.
const chartPartOf = "mining-pool"

// poolPod is a detected Bitcoin pool workload.
type poolPod struct {
	impl      model.PoolImpl
	namespace string
	pod       string
	container string
	node      string
	image     string
	phase     string
	running   bool
	start     time.Time
	apiPort   int
}

// fingerprint decides whether a pod is a Bitcoin solo pool and which
// implementation it runs. The container image is the discriminator — the chart
// names the workload "mining-pool" whichever pool is installed — while the chart
// labels and the stratum port only corroborate. A workload that looks like a pool
// but matches no known image is returned as ImplUnknown so the operator is told
// something was found rather than nothing.
func fingerprint(p *corev1.Pod, portOverride int) (poolPod, bool) {
	if len(p.Spec.Containers) == 0 {
		return poolPod{}, false
	}
	c := &p.Spec.Containers[0]
	image := strings.ToLower(c.Image)

	var impl model.PoolImpl
	switch {
	case strings.Contains(image, "public-pool"):
		impl = model.ImplPublicPool
	case strings.Contains(image, "ckpool"):
		impl = model.ImplCkpool
	case isChartPool(p, c):
		impl = model.ImplUnknown
	default:
		return poolPod{}, false
	}

	out := poolPod{
		impl:      impl,
		namespace: p.Namespace,
		pod:       p.Name,
		container: c.Name,
		node:      p.Spec.NodeName,
		image:     c.Image,
		phase:     string(p.Status.Phase),
		apiPort:   apiPort(c, portOverride),
	}
	for i := range p.Status.ContainerStatuses {
		cs := &p.Status.ContainerStatuses[i]
		switch {
		case cs.State.Running != nil:
			out.running = true
			out.start = cs.State.Running.StartedAt.Time
		case cs.State.Waiting != nil && cs.State.Waiting.Reason != "":
			out.phase = cs.State.Waiting.Reason
		case cs.State.Terminated != nil && cs.State.Terminated.Reason != "":
			out.phase = cs.State.Terminated.Reason
		}
	}
	return out, true
}

// isChartPool reports whether a pod carries the mining-pool chart's markers: its
// part-of label plus a stratum port. That is enough to say "a pool lives here",
// never enough to say which one.
func isChartPool(p *corev1.Pod, c *corev1.Container) bool {
	if p.Labels["app.kubernetes.io/part-of"] != chartPartOf {
		return false
	}
	for _, port := range c.Ports {
		if port.Name == "stratum" {
			return true
		}
	}
	return false
}

func apiPort(c *corev1.Container, override int) int {
	if override > 0 {
		return override
	}
	for _, p := range c.Ports {
		if p.Name == "api" {
			return int(p.ContainerPort)
		}
	}
	return defaultPublicPoolAPIPort
}

// detectPools finds every Bitcoin pool the credentials can see. It searches the
// namespace the operator named, or every namespace, narrowing to the miner's own
// namespace when a cluster-wide list is refused — so a token scoped to one
// namespace degrades to a smaller search rather than an error (FR-012). The
// returned scope says which it was, for the view to display honestly.
func (k *kubeClient) detectPools(ctx context.Context, cfg config.Config) (pods []poolPod, scope string, warn string) {
	scope = "all namespaces"
	if cfg.BTCNamespace != "" {
		scope = "namespace " + cfg.BTCNamespace
	}

	list, err := k.listPodsIn(ctx, cfg.BTCNamespace, cfg.BTCSelector)
	if err != nil && cfg.BTCNamespace == "" && errors.IsForbidden(err) {
		// Cluster-wide read denied: fall back to the namespace we know we can
		// read, and say so rather than reporting "no pool".
		denied := err
		if list, err = k.listPodsIn(ctx, k.ns, cfg.BTCSelector); err == nil {
			scope = "namespace " + k.ns + " (cluster-wide search denied)"
			warn = "Bitcoin pool search limited to namespace " + k.ns + ": " + denied.Error()
		}
	}
	if err != nil {
		return nil, "none", "Bitcoin pool discovery failed: " + err.Error()
	}

	for i := range list {
		if pp, ok := fingerprint(&list[i], cfg.BTCAPIPort); ok {
			pods = append(pods, pp)
		}
	}
	sort.Slice(pods, func(i, j int) bool {
		if pods[i].namespace != pods[j].namespace {
			return pods[i].namespace < pods[j].namespace
		}
		return pods[i].pod < pods[j].pod
	})
	return pods, scope, warn
}
