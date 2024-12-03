package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
)

const defaultScheduleInternal = time.Minute

// Trigger is an implementation of a glu.Trigger which runs promotions
// on a scheduled interval.
type Trigger struct {
	interval time.Duration
	options  []containers.Option[core.PhaseOptions]
}

// New creates a scheduled trigger for running automated promotion calls.
func New(opts ...containers.Option[Trigger]) *Trigger {
	trigger := &Trigger{
		interval: defaultScheduleInternal,
	}

	containers.ApplyAll(trigger, opts...)

	return trigger
}

// Run starts the scheduled calls of Promote on pipeline phases
// which match any configured target predicate.
func (t *Trigger) Run(ctx context.Context, edge core.Edge) {
	slog := slog.With("kind", edge.Kind(), "from", edge.From().Metadata.Name, "to", edge.To().Metadata.Name)
	slog.Debug("starting promotion schedule", "interval", t.interval)

	ticker := time.NewTicker(t.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := edge.Perform(ctx)
			if err != nil {
				slog.Error("edge perform", "error", err)
			}

			slog.Info("edge perform succeeded", "annotations", result.Annotations)
		}
	}
}

// WithInterval sets the interval on a schedule
func WithInterval(d time.Duration) containers.Option[Trigger] {
	return func(t *Trigger) {
		t.interval = d
	}
}
