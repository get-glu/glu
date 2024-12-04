package edges

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/get-glu/glu/pkg/core"
	"github.com/get-glu/glu/pkg/core/typed"
)

var _ core.Edge = (*PromotionEdge[core.Resource])(nil)

// ErrSkipped is returned when an edge skips performing because the operation
// would be a no-op.
var ErrSkipped = errors.New("skipped performing")

// PromotionEdge is a type edge implementation which supports promoting
// from a source phase to a destination phase.
type PromotionEdge[R core.Resource] struct {
	logger *slog.Logger
	from   typed.Phase[R]
	to     typed.UpdatablePhase[R]
}

func Promotes[R core.Resource](from typed.Phase[R], to typed.UpdatablePhase[R]) *PromotionEdge[R] {
	return &PromotionEdge[R]{
		logger: slog.With("from", from.Descriptor().String(), "to", to.Descriptor().String()),
		from:   from,
		to:     to,
	}
}

func (s *PromotionEdge[R]) Kind() string {
	return typed.KindPromotion
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
func (s *PromotionEdge[R]) Perform(ctx context.Context) (r *core.Result, err error) {
	s.logger.Debug("edge perform started")
	defer func() {
		var args []any
		if err != nil {
			err = fmt.Errorf("promoting from %s to %s: %w", s.from.Descriptor().Metadata.Name, s.to.Descriptor().Metadata.Name, err)
			args = append(args, "error", err)
		}

		s.logger.Debug("edge perform finished", args...)
	}()

	from, synced, err := s.synced(ctx)
	if err != nil {
		return r, err
	}

	if synced {
		s.logger.Debug("skipping promotion", "reason", "UpToDate")
		return nil, ErrSkipped
	}

	return s.to.Update(ctx, from, typed.UpdateWithKind(typed.KindPromotion))
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
