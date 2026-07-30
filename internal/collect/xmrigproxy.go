package collect

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// proxyGet fetches path (e.g. "1/summary") from a pod's HTTP server via the
// API-server pod proxy subresource — a read, requiring only pods/proxy access
// (Constitution II). No local port-forward is set up.
func (k *kubeClient) proxyGet(ctx context.Context, pod, path string) ([]byte, error) {
	name := fmt.Sprintf("%s:%d", pod, k.apiPort)
	return k.cs.CoreV1().RESTClient().Get().
		Namespace(k.ns).
		Resource("pods").
		SubResource("proxy").
		Name(name).
		Suffix(path).
		DoRaw(ctx)
}

// podLogs fetches the last tail lines of a pod container's logs.
func (k *kubeClient) podLogs(ctx context.Context, pod, container string, tail int64) ([]byte, error) {
	opts := &corev1.PodLogOptions{TailLines: &tail}
	if container != "" {
		opts.Container = container
	}
	return k.cs.CoreV1().Pods(k.ns).GetLogs(pod, opts).DoRaw(ctx)
}
