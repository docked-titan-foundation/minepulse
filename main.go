// Command minepulse is a read-only terminal scope for the monero-idle-miner
// DaemonSet: it shows, continuously, how Monero mining is going across a
// Kubernetes cluster.
package main

import "github.com/docked-titan-foundation/minepulse/cmd"

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(version)
}
