package collect

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// proxyGet fetches path (e.g. "1/summary") from a miner pod's HTTP server in the
// configured namespace, on the XMRig API port.
func (k *kubeClient) proxyGet(ctx context.Context, pod, path string) ([]byte, error) {
	return k.proxyGetIn(ctx, k.ns, pod, k.apiPort, path)
}

// proxyGetIn fetches path from any pod's HTTP server via the API-server pod proxy
// subresource — a read, requiring only pods/proxy access (Constitution II). No
// local port-forward is set up. Namespace and port are explicit because the
// Bitcoin pool lives wherever the operator installed it, on its own port.
func (k *kubeClient) proxyGetIn(ctx context.Context, ns, pod string, port int, path string) ([]byte, error) {
	name := fmt.Sprintf("%s:%d", pod, port)
	return k.cs.CoreV1().RESTClient().Get().
		Namespace(ns).
		Resource("pods").
		SubResource("proxy").
		Name(name).
		Suffix(path).
		DoRaw(ctx)
}

// podLogs fetches the last tail lines of a miner pod container's logs.
func (k *kubeClient) podLogs(ctx context.Context, pod, container string, tail int64) ([]byte, error) {
	return k.podLogsIn(ctx, k.ns, pod, container, tail)
}

// podLogsIn fetches the last tail lines of any pod container's logs.
func (k *kubeClient) podLogsIn(ctx context.Context, ns, pod, container string, tail int64) ([]byte, error) {
	opts := &corev1.PodLogOptions{TailLines: &tail}
	if container != "" {
		opts.Container = container
	}
	return k.cs.CoreV1().Pods(ns).GetLogs(pod, opts).DoRaw(ctx)
}
