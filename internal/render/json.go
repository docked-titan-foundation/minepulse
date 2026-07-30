package render

import (
	"encoding/json"
	"io"

	"github.com/docked-titan-foundation/minepulse/internal/model"
)

// JSON writes a snapshot as a single line of JSON (jsonl) — the stable
// machine-readable output contract.
func JSON(w io.Writer, s *model.Snapshot) error {
	enc := json.NewEncoder(w)
	return enc.Encode(s) // Encode appends a newline
}
