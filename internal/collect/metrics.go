package collect

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nodeCPUUsed returns used CPU (millicores) per node from metrics-server.
func (k *kubeClient) nodeCPUUsed(ctx context.Context) (map[string]int64, error) {
	list, err := k.metrics.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(list.Items))
	for i := range list.Items {
		n := &list.Items[i]
		out[n.Name] = n.Usage.Cpu().MilliValue()
	}
	return out, nil
}

// podCPUUsed returns used CPU (millicores) per pod in the namespace, summed
// across the pod's containers.
func (k *kubeClient) podCPUUsed(ctx context.Context) (map[string]int64, error) {
	list, err := k.metrics.MetricsV1beta1().PodMetricses(k.ns).List(ctx, metav1.ListOptions{LabelSelector: k.selector})
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(list.Items))
	for i := range list.Items {
		p := &list.Items[i]
		var sum int64
		for j := range p.Containers {
			sum += p.Containers[j].Usage.Cpu().MilliValue()
		}
		out[p.Name] = sum
	}
	return out, nil
}
