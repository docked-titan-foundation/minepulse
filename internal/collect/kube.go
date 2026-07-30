package collect

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/docked-titan-foundation/minepulse/internal/config"
)

// kubeClient wraps the read-only cluster access minepulse needs. It constructs
// only read verbs; no write client is ever built (Constitution II).
type kubeClient struct {
	cs       *kubernetes.Clientset
	metrics  *metricsv.Clientset
	ns       string
	selector string
	apiPort  int
}

func newKubeClient(cfg config.Config) (*kubeClient, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cfg.Kubeconfig != "" {
		rules.ExplicitPath = cfg.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if cfg.Context != "" {
		overrides.CurrentContext = cfg.Context
	}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	restCfg, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}
	mc, err := metricsv.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("metrics client: %w", err)
	}
	return &kubeClient{cs: cs, metrics: mc, ns: cfg.Namespace, selector: cfg.Selector, apiPort: 8080}, nil
}

// minerPod is the pod-level view of one miner instance.
type minerPod struct {
	node      string
	pod       string
	container string
	image     string
	phase     string // display phase (container waiting/terminated reason when applicable)
	running   bool
	restarts  int32
	start     time.Time
	wallet    string // parsed from the container's -u/--user arg
	pool      string // parsed from the container's -o/--url arg
}

// listMiners returns one minerPod per pod matching the selector in the namespace.
func (k *kubeClient) listMiners(ctx context.Context) ([]minerPod, error) {
	pods, err := k.cs.CoreV1().Pods(k.ns).List(ctx, metav1.ListOptions{LabelSelector: k.selector})
	if err != nil {
		return nil, err
	}
	out := make([]minerPod, 0, len(pods.Items))
	for i := range pods.Items {
		p := &pods.Items[i]
		mp := minerPod{node: p.Spec.NodeName, pod: p.Name, phase: string(p.Status.Phase)}
		if len(p.Spec.Containers) > 0 {
			c := p.Spec.Containers[0]
			mp.container = c.Name
			mp.image = c.Image
			mp.wallet, mp.pool = parseMinerArgs(append(append([]string{}, c.Command...), c.Args...))
		}
		for j := range p.Status.ContainerStatuses {
			cs := &p.Status.ContainerStatuses[j]
			mp.restarts += cs.RestartCount
			switch {
			case cs.State.Running != nil:
				mp.running = true
				mp.start = cs.State.Running.StartedAt.Time
			case cs.State.Waiting != nil && cs.State.Waiting.Reason != "":
				mp.phase = cs.State.Waiting.Reason
			case cs.State.Terminated != nil && cs.State.Terminated.Reason != "":
				mp.phase = cs.State.Terminated.Reason
			}
		}
		out = append(out, mp)
	}
	return out, nil
}

// nodeCPUCapacity returns allocatable CPU (millicores) for the given nodes.
func (k *kubeClient) nodeCPUCapacity(ctx context.Context, nodes []string) map[string]int64 {
	out := map[string]int64{}
	list, err := k.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return out
	}
	want := map[string]bool{}
	for _, n := range nodes {
		want[n] = true
	}
	for i := range list.Items {
		n := &list.Items[i]
		if want[n.Name] {
			out[n.Name] = n.Status.Allocatable.Cpu().MilliValue()
		}
	}
	return out
}

// parseMinerArgs extracts the wallet (-u/--user) and pool (-o/--url) from an
// XMRig argv, handling both "flag value" and "flag=value" forms.
func parseMinerArgs(argv []string) (wallet, pool string) {
	get := func(i int) string {
		if i+1 < len(argv) {
			return argv[i+1]
		}
		return ""
	}
	for i, a := range argv {
		switch {
		case a == "-u" || a == "--user":
			wallet = get(i)
		case strings.HasPrefix(a, "--user="):
			wallet = strings.TrimPrefix(a, "--user=")
		case a == "-o" || a == "--url":
			pool = get(i)
		case strings.HasPrefix(a, "--url="):
			pool = strings.TrimPrefix(a, "--url=")
		}
	}
	return wallet, pool
}
