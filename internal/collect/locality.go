package collect

import (
	"context"
	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// clusterNet is the set of addresses that belong to this cluster: every Pod IP
// and Service ClusterIP minepulse was allowed to read, each remembered with the
// object it came from.
//
// This exists because "looks private" is not "is in this cluster". A pool on the
// operator's NAS answers on 192.168.1.40 and is emphatically external; reporting
// it as internal would misstate where their hashrate goes. Range inspection is
// kept only as the fallback for when the cluster cannot be read at all, and it
// says so when it is used (FR-003, FR-004).
type clusterNet struct {
	owner  map[string]string // IP → the object holding it
	readOK bool              // whether the object listing actually succeeded
}

// scanClusterNet lists Pod IPs and Service ClusterIPs across the namespaces
// minepulse can see. A failure is not fatal: it returns a clusterNet that says
// it could not read, and classification degrades to ranges (Constitution III).
func (k *kubeClient) scanClusterNet(ctx context.Context) clusterNet {
	cn := clusterNet{owner: map[string]string{}}

	svcs, svcErr := k.cs.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if svcErr == nil {
		for i := range svcs.Items {
			s := &svcs.Items[i]
			for _, ip := range append([]string{s.Spec.ClusterIP}, s.Spec.ClusterIPs...) {
				if ip != "" && ip != "None" {
					cn.owner[ip] = fmt.Sprintf("Service %s/%s", s.Namespace, s.Name)
				}
			}
			for _, ip := range s.Spec.ExternalIPs {
				cn.owner[ip] = fmt.Sprintf("Service %s/%s", s.Namespace, s.Name)
			}
			for _, ing := range s.Status.LoadBalancer.Ingress {
				if ing.IP != "" {
					cn.owner[ing.IP] = fmt.Sprintf("Service %s/%s", s.Namespace, s.Name)
				}
			}
		}
	}

	pods, podErr := k.cs.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if podErr == nil {
		for i := range pods.Items {
			p := &pods.Items[i]
			for _, ip := range p.Status.PodIPs {
				if ip.IP != "" {
					cn.owner[ip.IP] = fmt.Sprintf("Pod %s/%s", p.Namespace, p.Name)
				}
			}
			if p.Status.PodIP != "" {
				cn.owner[p.Status.PodIP] = fmt.Sprintf("Pod %s/%s", p.Namespace, p.Name)
			}
		}
	}

	// Either listing succeeding is enough to make a positive match meaningful.
	cn.readOK = svcErr == nil || podErr == nil
	return cn
}

// classify decides whether ip belongs to this cluster, and says on what evidence.
//
// A match is the strong answer and is always trusted. A miss is only conclusive
// when the objects were readable — otherwise the honest answer is that we do not
// know, softened to a range guess that names itself as one.
func (c clusterNet) classify(ip string) (model.Locality, string) {
	if ip == "" {
		return model.LocalityUnknown, "no address reported"
	}
	if owner, ok := c.owner[ip]; ok {
		return model.LocalityInternal, "matched " + owner
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return model.LocalityUnknown, "unparseable address"
	}
	if c.readOK {
		if isPrivate(parsed) {
			// Private, but not one of this cluster's objects: something else on
			// the network. External to the cluster, which is the question asked.
			return model.LocalityExternal, "private address, no cluster object matches"
		}
		return model.LocalityExternal, "public address, no cluster object matches"
	}
	// Could not read the cluster's objects — fall back, and say so (FR-003).
	if isPrivate(parsed) {
		return model.LocalityUnknown, "cannot read cluster objects; address is private"
	}
	return model.LocalityExternal, "cannot read cluster objects; address is public"
}

// isPrivate reports whether ip is in a range that cannot be routed on the public
// internet — RFC1918, CGNAT, loopback and link-local.
func isPrivate(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || inCGNAT(ip)
}

// inCGNAT covers 100.64.0.0/10, which several CNIs and tailnets hand out and
// net.IP.IsPrivate does not count.
func inCGNAT(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}
