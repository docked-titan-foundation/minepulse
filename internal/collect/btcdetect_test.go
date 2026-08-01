package collect

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func pod(image string, labels map[string]string, ports ...corev1.ContainerPort) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "mining-pool-abc", Namespace: "bitcoin", Labels: labels},
		Spec: corev1.PodSpec{
			NodeName:   "orion",
			Containers: []corev1.Container{{Name: "pool", Image: image, Ports: ports}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

// Classification is evidence-based: the image says which implementation runs, and
// a pool-shaped workload that matches nothing is reported as unknown, never
// guessed (FR-006).
func TestFingerprint(t *testing.T) {
	chartLabels := map[string]string{"app.kubernetes.io/part-of": "mining-pool"}
	apiPort := corev1.ContainerPort{Name: "api", ContainerPort: 3334}
	stratum := corev1.ContainerPort{Name: "stratum", ContainerPort: 3333}

	tests := []struct {
		name     string
		p        corev1.Pod
		wantImpl model.PoolImpl
		wantPool bool
		wantPort int
	}{
		{
			name:     "public-pool by image",
			p:        pod("ghcr.io/docked-titan-foundation/public-pool:v1.2.3", chartLabels, apiPort, stratum),
			wantImpl: model.ImplPublicPool, wantPool: true, wantPort: 3334,
		},
		{
			name:     "public-pool on a non-default api port",
			p:        pod("ghcr.io/x/public-pool@sha256:abc", nil, corev1.ContainerPort{Name: "api", ContainerPort: 8080}),
			wantImpl: model.ImplPublicPool, wantPool: true, wantPort: 8080,
		},
		{
			name:     "public-pool with no named api port falls back to 3334",
			p:        pod("ghcr.io/x/public-pool:v1", nil, stratum),
			wantImpl: model.ImplPublicPool, wantPool: true, wantPort: 3334,
		},
		{
			name:     "ckpool by image",
			p:        pod("ghcr.io/docked-titan-foundation/ckpool:v1.0.0", chartLabels, stratum),
			wantImpl: model.ImplCkpool, wantPool: true,
		},
		{
			name:     "chart workload with an unrecognized image",
			p:        pod("ghcr.io/someone/mystery-pool:v9", chartLabels, stratum),
			wantImpl: model.ImplUnknown, wantPool: true,
		},
		{
			name:     "unrelated pod is not a pool",
			p:        pod("metal3d/xmrig:6.20.0", map[string]string{"app.kubernetes.io/name": "monero-idle-miner"}),
			wantPool: false,
		},
		{
			name:     "stratum port alone is not enough evidence",
			p:        pod("nginx:1.27", nil, corev1.ContainerPort{Name: "http", ContainerPort: 80}),
			wantPool: false,
		},
	}

	for _, tt := range tests {
		got, ok := fingerprint(&tt.p, 0)
		if ok != tt.wantPool {
			t.Errorf("%s: detected = %v, want %v", tt.name, ok, tt.wantPool)
			continue
		}
		if !ok {
			continue
		}
		if got.impl != tt.wantImpl {
			t.Errorf("%s: impl = %q, want %q", tt.name, got.impl, tt.wantImpl)
		}
		if tt.wantPort != 0 && got.apiPort != tt.wantPort {
			t.Errorf("%s: apiPort = %d, want %d", tt.name, got.apiPort, tt.wantPort)
		}
	}
}

// An explicit --btc-api-port beats what the pod declares.
func TestFingerprintPortOverride(t *testing.T) {
	p := pod("ghcr.io/x/public-pool:v1", nil, corev1.ContainerPort{Name: "api", ContainerPort: 3334})
	got, ok := fingerprint(&p, 9999)
	if !ok || got.apiPort != 9999 {
		t.Errorf("apiPort = %d (detected %v), want the 9999 override", got.apiPort, ok)
	}
}
