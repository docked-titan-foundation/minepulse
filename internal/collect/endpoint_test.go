package collect

import (
	"strings"
	"testing"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

func node(name, pool, ip string) model.NodeStatus {
	return model.NodeStatus{
		Node:   name,
		Mining: &model.MiningStats{Pool: pool, PoolIP: ip, Connected: true},
	}
}

// The line names the pool most miners are on — but a node that has fallen back
// to the donate pool must not be averaged away, because catching exactly that is
// what the tool is for (FR-007).
func TestMoneroEndpointMarksDivergence(t *testing.T) {
	nodes := []model.NodeStatus{
		node("andromeda", "pool.supportxmr.com:443", "116.202.180.221"),
		node("cygnus", "pool.supportxmr.com:443", "116.202.180.221"),
		node("orion", "donate.v2.xmrig.com:3333", "178.128.242.134"),
	}
	e := moneroEndpoint(nodes, clusterNet{owner: map[string]string{}, readOK: true}, "pool.supportxmr.com:443")
	if e == nil {
		t.Fatal("no endpoint")
	}
	if e.URL != "pool.supportxmr.com:443" {
		t.Errorf("URL = %q, want the majority pool", e.URL)
	}
	if !e.Diverged {
		t.Error("miners are on two different pools; Diverged must be set")
	}
	if e.Brand != "SupportXMR" || e.Mode != model.ModeShared {
		t.Errorf("brand/mode = %q/%q", e.Brand, e.Mode)
	}

	// All on one pool: no marker.
	same := moneroEndpoint(nodes[:2], clusterNet{readOK: true}, "pool.supportxmr.com:443")
	if same.Diverged {
		t.Error("all miners agree; Diverged must not be set")
	}
}

// Nothing connected yet still names the configured pool — there is simply no
// resolved address to locate it by.
func TestMoneroEndpointBeforeAnyConnection(t *testing.T) {
	e := moneroEndpoint(nil, clusterNet{readOK: true}, "pool.supportxmr.com:443")
	if e == nil || e.URL != "pool.supportxmr.com:443" {
		t.Fatalf("endpoint = %+v", e)
	}
	if e.Locality != model.LocalityUnknown || e.IP != "" {
		t.Errorf("no connection means no locality claim, got %q/%q", e.Locality, e.IP)
	}
	if moneroEndpoint(nil, clusterNet{readOK: true}, "") != nil {
		t.Error("no miners and no configured pool is no endpoint at all")
	}
}

// "internal" is a claim about cluster objects, not about address ranges. A pool
// on the operator's NAS is private and still external, and saying otherwise
// would misreport where their hashrate goes (FR-002/FR-004).
func TestClassifyPrefersObjectsOverRanges(t *testing.T) {
	cn := clusterNet{
		readOK: true,
		owner:  map[string]string{"10.43.7.12": "Service bitcoin/mining-pool"},
	}
	for _, tc := range []struct {
		ip        string
		want      model.Locality
		basisHas  string
		skipBasis bool
	}{
		{"10.43.7.12", model.LocalityInternal, "Service bitcoin/mining-pool", false},
		// Private, but nothing in the cluster owns it: the NAS case.
		{"192.168.1.40", model.LocalityExternal, "no cluster object matches", false},
		{"116.202.180.221", model.LocalityExternal, "public address", false},
		{"", model.LocalityUnknown, "no address", false},
		{"not-an-ip", model.LocalityUnknown, "unparseable", false},
	} {
		got, basis := cn.classify(tc.ip)
		if got != tc.want {
			t.Errorf("classify(%q) = %q, want %q (basis %q)", tc.ip, got, tc.want, basis)
		}
		if !tc.skipBasis && !strings.Contains(basis, tc.basisHas) {
			t.Errorf("classify(%q) basis = %q, want it to mention %q", tc.ip, basis, tc.basisHas)
		}
	}
}

// With the cluster unreadable, a private address must not be reported as
// internal on range evidence alone — the weaker test names itself (FR-003).
func TestClassifyDegradesWithoutClusterReads(t *testing.T) {
	cn := clusterNet{owner: map[string]string{}, readOK: false}

	got, basis := cn.classify("10.43.7.12")
	if got != model.LocalityUnknown {
		t.Errorf("unreadable cluster + private address = %q, want unknown", got)
	}
	if !strings.Contains(basis, "cannot read cluster objects") {
		t.Errorf("the fallback must say it is one, got %q", basis)
	}

	if got, _ := cn.classify("116.202.180.221"); got != model.LocalityExternal {
		t.Errorf("a public address is external regardless of cluster reads, got %q", got)
	}
}

// public-pool is solo by design; ckpool ships both ways and only its image says
// which, so it must not inherit this repo's solo-mining assumption (FR-006).
func TestImplMode(t *testing.T) {
	for _, tc := range []struct {
		impl  model.PoolImpl
		image string
		want  model.MiningMode
	}{
		{model.ImplPublicPool, "ghcr.io/benjamin-wilson/public-pool:latest", model.ModeSolo},
		{model.ImplCkpool, "ghcr.io/example/ckpool-solo:1.0", model.ModeSolo},
		{model.ImplCkpool, "ghcr.io/example/ckpool:1.0", model.ModeUnknown},
		{model.ImplUnknown, "some/image", model.ModeUnknown},
	} {
		if got := implMode(poolPod{impl: tc.impl, image: tc.image}); got != tc.want {
			t.Errorf("implMode(%s, %q) = %q, want %q", tc.impl, tc.image, got, tc.want)
		}
	}
}
