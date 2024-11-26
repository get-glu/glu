package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/get-glu/glu"
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
func (t *Trigger) Run(ctx context.Context, p glu.Pipelines) {
	slog.Debug("starting promotion schedule", "interval", t.interval)

	ticker := time.NewTicker(t.interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, pipeline := range p.Pipelines() {
				for phase := range pipeline.Phases(t.options...) {
					if err := phase.Promote(ctx); err != nil {
						slog.Error("promoting resource", "name", phase.Metadata().Name, "error", err)
					}
				}
			}

		}
	}
}

// WithInterval sets the interval on a schedule
func WithInterval(d time.Duration) containers.Option[Trigger] {
	return func(t *Trigger) {
		t.interval = d
	}
}

// MatchesPhase sets a match condition which matches a specific phase
func MatchesPhase(c core.Phase) containers.Option[Trigger] {
	return func(t *Trigger) {
		t.options = append(t.options, core.IsPhase(c))
	}
}

// MatchesName sets a match condition which matches a specific phase name
func MatchesName(name string) containers.Option[Trigger] {
	return func(t *Trigger) {
		t.options = append(t.options, core.HasName(name))
	}
}

// MatchesLabel sets a match condition which matches any phase with the provided label
func MatchesLabel(k, v string) containers.Option[Trigger] {
	return func(t *Trigger) {
		t.options = append(t.options, core.HasLabel(k, v))
	}
}
