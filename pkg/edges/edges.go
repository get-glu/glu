package edges

import (
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
)

// PromotionEdge is an edge implementation which supports promoting
// from a source phase to a destination phase.
type PromotionEdge struct {
	logger *slog.Logger
	from   core.Phase
	to     core.Phase
}

func Promotes(from core.Phase, to core.Phase) *PromotionEdge {
	return &PromotionEdge{
		logger: slog.With("from", from.Descriptor().String(), "to", to.Descriptor().String()),
		from:   from,
		to:     to,
	}
}

func (s *PromotionEdge) From() core.Descriptor {
	return s.from.Descriptor()
}

func (s *PromotionEdge) To() core.Descriptor {
	return s.to.Descriptor()
}
