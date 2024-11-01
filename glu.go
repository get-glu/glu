package glu

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/flipt-io/glu/pkg/config"
	"github.com/flipt-io/glu/pkg/containers"
	"github.com/flipt-io/glu/pkg/core"
	"github.com/flipt-io/glu/pkg/credentials"
	"github.com/flipt-io/glu/pkg/repository"
)

type Metadata = core.Metadata

type Pipeline struct {
	*core.Pipeline

	ctx   context.Context
	conf  *config.Config
	creds *credentials.CredentialSource

	scheduled []scheduled
}

type scheduled struct {
	core.Reconciler

	interval time.Duration
}

func NewPipeline(ctx context.Context, name string) (*Pipeline, error) {
	conf, err := config.ReadFromPath("glu.yaml")
	if err != nil {
		return nil, err
	}

	return &Pipeline{
		Pipeline: core.NewPipeline(ctx, name),
		ctx:      ctx,
		conf:     conf,
		creds:    credentials.New(conf.Credentials),
	}, nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	for _, sch := range p.scheduled {
		wg.Add(1)
		go func(sch scheduled) {
			defer wg.Done()

			ticker := time.NewTicker(sch.interval)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := sch.Reconcile(ctx); err != nil {
						meta := sch.Metadata()
						slog.Error("reconciling resource", "phase", meta.Phase, "name", meta.Name, "error", err)
					}
				}
			}
		}(sch)
	}

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		wg.Wait()
	}()

	<-ctx.Done()

	select {
	case <-time.After(15 * time.Second):
		return errors.New("timedout waiting on shutdown of schedules")
	case <-finished:
		return ctx.Err()
	}
}

func (p *Pipeline) ScheduleReconcile(r core.Reconciler, interval time.Duration) {
	p.scheduled = append(p.scheduled, scheduled{r, interval})
}

type RepositoryOptions struct {
	enableProposals bool
}

func EnableProposals(o *RepositoryOptions) {
	o.enableProposals = true
}

func (p *Pipeline) NewRepository(name string, opts ...containers.Option[RepositoryOptions]) (core.Repository, error) {
	var options RepositoryOptions
	containers.ApplyAll(&options, opts...)

	conf, ok := p.conf.Repositories[name]
	if !ok {
		return nil, fmt.Errorf("repository %q: configuration not found", name)
	}

	return repository.NewGitRepository(p.ctx, conf, p.creds, name, options.enableProposals)
}
