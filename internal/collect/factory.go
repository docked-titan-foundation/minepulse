package collect

import "github.com/docked-titan-foundation/minepulse/internal/config"

// New builds the Source for a run: the synthetic source when --mock is set,
// otherwise the live cluster source.
func New(cfg config.Config) (Source, error) {
	if cfg.Mock {
		return NewMock(cfg), nil
	}
	return newClusterSource(cfg)
}
