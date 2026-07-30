package xmrig

import (
	"encoding/json"
	"fmt"
)

// Backend is one entry of GET /2/backends. For the CPU backend, the length of
// the threads array is the number of mining threads XMRig actually started.
type Backend struct {
	Type    string            `json:"type"`
	Enabled bool              `json:"enabled"`
	Threads []json.RawMessage `json:"threads"`
}

// ParseBackends decodes a /2/backends payload.
func ParseBackends(b []byte) ([]Backend, error) {
	var out []Backend
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse xmrig backends: %w", err)
	}
	return out, nil
}

// ActiveCPUThreads returns the number of started CPU mining threads, or -1 if
// the CPU backend is absent or disabled.
func ActiveCPUThreads(backends []Backend) int {
	for _, b := range backends {
		if b.Type == "cpu" && b.Enabled {
			return len(b.Threads)
		}
	}
	return -1
}
