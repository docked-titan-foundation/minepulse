// Package btc parses the Bitcoin solo-mining pools minepulse observes:
// public-pool's JSON stats API and ckpool's periodic status records.
//
// Like internal/xmrig and internal/pool, everything here is pure: it takes
// bytes and returns model types, holds no cluster client, and opens no
// connection, so it is fixture-tested end to end (Constitution IV). Fetching
// those bytes — through the API server's pod proxy or a pod's log stream — is
// internal/collect's job.
package btc
