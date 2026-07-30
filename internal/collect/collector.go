// Package collect turns cluster and pool state into model.Snapshots.
package collect

import (
	"context"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// Source produces one Snapshot per call. It is created once and may hold state
// across calls (e.g. per-node history rings). Implementations MUST be read-only
// toward whatever they observe (Constitution II) and MUST degrade gracefully:
// a failure of one underlying source becomes a warning or an "unavailable"
// field, never an error that aborts the whole snapshot (Constitution III).
type Source interface {
	// Gather returns the current snapshot. It returns an error only when it can
	// produce nothing at all (e.g. no cluster access); partial data is normal
	// and is conveyed via nil fields, sentinel values, and Snapshot.Warnings.
	Gather(ctx context.Context) (*model.Snapshot, error)

	// Close releases any resources held by the source.
	Close() error
}
