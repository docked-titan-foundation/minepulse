# Quickstart: minepulse

## Build

```bash
mise install            # pins go + toolchain
go build -o bin/minepulse .
```

## See it with no cluster (mock)

```bash
minepulse watch --mock
```

Renders the full TUI with synthetic nodes — hashrate, threads, shares, a CPU/free
sparkline per node, and a pool panel. Ctrl-C / `q` to quit. This is also what CI and
`go test`-adjacent demos use (Constitution III / SC-005).

## Watch the real cluster

```bash
minepulse watch -n monero-idle-miner
```

Defaults: selector `app.kubernetes.io/name=monero-idle-miner`, interval 3s, output
`tui` on a terminal. Needs a kubeconfig with the read access in `deploy/rbac.yaml`.

Useful flags:

```bash
minepulse watch -n monero-idle-miner \
  --interval 5s \
  --output stream \          # compact text per tick (pipe / background / agent)
  --wallet <XMR-address> \   # override auto-detected wallet
  --no-pool \                # skip pool-side earnings
  --xmrig-api auto|on|off    # force the miner data source

minepulse snapshot -n monero-idle-miner --output json   # one JSON snapshot, then exit
```

## Prove the dynamic idle-CPU behaviour (User Story 2)

In one terminal:

```bash
minepulse watch -n monero-idle-miner
```

In another, drop a CPU-hungry Guaranteed pod on a mining node, watch that node's
miner-CPU dip and free-CPU rise in the sparkline, then delete it and watch it recover:

```bash
kubectl run cpu-hog --image=busybox --restart=Never \
  --overrides='{"spec":{"nodeName":"<a-mining-node>"}}' \
  --requests=cpu=2 --limits=cpu=2 -- sh -c 'while :; do :; done & while :; do :; done & wait'
# ...observe the dip...
kubectl delete pod cpu-hog     # ...observe recovery
```

## Richest miner data (optional)

Detailed hashrate/threads/shares come from the XMRig HTTP API when the miner exposes it
(`httpApi.enabled: true` in `apps/monero-idle-miner/values.yaml`). Without it, minepulse
falls back to parsing miner logs and marks the node's stats source as `logs`.
