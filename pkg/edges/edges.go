package edges

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
)

// Source is an interface around storage for resources.
type Phase[R core.Resource] interface {
	core.Phase
	GetResource(_ context.Context) (R, error)
}

// UpdatableSource is a source through which the phase can promote resources to new versions
type UpdatablePhase[R core.Resource] interface {
	Phase[R]
	Update(_ context.Context, to R) (map[string]string, error)
}

var _ core.Edge = (*PromotionEdge[core.Resource])(nil)

type PromotionEdge[R core.Resource] struct {
	logger *slog.Logger
	from   Phase[R]
	to     UpdatablePhase[R]
}

func Promotes[R core.Resource](from Phase[R], to UpdatablePhase[R]) *PromotionEdge[R] {
	return &PromotionEdge[R]{
		logger: slog.With("from", from.Descriptor().String(), "to", to.Descriptor().String()),
		from:   from,
		to:     to,
	}
}

func (s *PromotionEdge[R]) Kind() string {
	return "promotion"
}

func (s *PromotionEdge[R]) From() core.Descriptor {
	return s.from.Descriptor()
}

func (s *PromotionEdge[R]) To() core.Descriptor {
	return s.to.Descriptor()
}

// Perform causes a promotion from a dependent to a target phase.
// The phase fetches both its current resource state, and that of the promotion source phase.
// If the resources differ, then the phase updates its source to match the promoted version.
func (s *PromotionEdge[R]) Perform(ctx context.Context) (r core.Result, err error) {
	s.logger.Debug("Promotion started")
	defer func() {
		s.logger.Debug("Promotion finished")
		if err != nil {
			err = fmt.Errorf("promoting from %s to %s: %w", s.from.Descriptor().Metadata.Name, s.to.Descriptor().Metadata.Name, err)
		}
	}()

	from, synced, err := s.synced(ctx)
	if err != nil {
		return r, err
	}

	if synced {
		s.logger.Debug("skipping promotion", "reason", "UpToDate")
		return r, nil
	}

	if r.Annotations, err = s.to.Update(ctx, from); err != nil {
		return r, err
	}

	return r, nil
}

func (s *PromotionEdge[R]) CanPerform(ctx context.Context) (bool, error) {
	_, synced, err := s.synced(ctx)
	return synced, err
}

func (s *PromotionEdge[R]) synced(ctx context.Context) (from R, synced bool, err error) {
	from, err = s.from.GetResource(ctx)
	if err != nil {
		return from, false, err
	}

	fromDigest, err := from.Digest()
	if err != nil {
		return from, false, err
	}

	to, err := s.to.GetResource(ctx)
	if err != nil {
		return from, false, err
	}

	toDigest, err := to.Digest()
	if err != nil {
		return from, false, err
	}

	if fromDigest == toDigest {
		return from, true, nil
	}

	return from, false, nil
}
