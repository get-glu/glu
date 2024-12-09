package triggers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/get-glu/glu/pkg/core"
)

// Edge returns a triggerable edge implementations which on a call to RunTriggers
// runs all the provided triggers passing the edge as an argument.
func Edge(e core.Edge, triggers ...Trigger) core.Edge {
	if len(triggers) == 0 {
		return e
	}

	return triggerableEdge{
		Edge:     e,
		triggers: triggers,
	}
}

// Trigger is a type with a single method Run which takes a context
// and an Edge implementation.
// The triggers package expects Run to block until the context provided
// is cancelled.
type Trigger interface {
	Run(context.Context, core.Edge)
}

type triggerableEdge struct {
	core.Edge

	triggers []Trigger
}

// RunTriggers runs all configured triggers passing them the decorated Edge.
// It blocks until all triggers have completed.
// Shutdown is signalled via context cancellation.
func (t triggerableEdge) RunTriggers(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("running triggers: %w", err)
		}
	}()

	ctx, cancel := context.WithCancel(ctx)

	slog := slog.With("kind", t.Kind(),
		"from", t.From().Metadata.Name,
		"to", t.To().Metadata.Name,
	)

	var wg sync.WaitGroup
	for _, trigger := range t.triggers {
		wg.Add(1)
		go func(trigger Trigger) {
			defer wg.Done()

			trigger.Run(ctx, t.Edge)
		}(trigger)
	}

	finished := make(chan struct{})
	go func() {
		defer func() {
			cancel()

			close(finished)
		}()

		wg.Wait()
	}()

	<-ctx.Done()

	select {
	case <-time.After(15 * time.Second):
		return errors.New("timedout waiting on shutdown of triggers")
	case <-finished:
		slog.Info("edge triggers finished")

		return ctx.Err()
	}
}
