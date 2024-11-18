package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/get-glu/glu/pkg/core"
)

type System interface {
	GetPipeline(name string) (core.Pipeline, error)
	Pipelines() iter.Seq2[string, core.Pipeline]
}

func Run(ctx context.Context, s System, args ...string) error {
	switch args[1] {
	case "inspect":
		return inspect(ctx, s, args[2:]...)
	case "promote":
		return promote(ctx, s, args[2:]...)
	default:
		return fmt.Errorf("unexpected command %q (expected one of [inspect promote])", args[1])
	}
}

func inspect(ctx context.Context, s System, args ...string) (err error) {
	wr := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() {
		if ferr := wr.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	if len(args) == 0 {
		fmt.Fprintln(wr, "NAME")
		for name := range s.Pipelines() {
			fmt.Fprintln(wr, name)
		}
		return nil
	}

	pipeline, err := s.GetPipeline(args[0])
	if err != nil {
		return err
	}

	if len(args) == 1 {
		fmt.Fprintln(wr, "NAME\tDEPENDS_ON")
		deps := pipeline.Dependencies()
		for phase := range pipeline.Phases() {
			dependsName := ""
			if depends, ok := deps[phase]; ok && depends != nil {
				dependsName = depends.Metadata().Name
			}

			fmt.Fprintf(wr, "%s\t%s\n", phase.Metadata().Name, dependsName)
		}
		return nil
	}

	phase, err := pipeline.PhaseByName(args[1])
	if err != nil {
		return err
	}

	inst, err := phase.Get(ctx)
	if err != nil {
		return err
	}

	var extraFields [][2]string
	fields, ok := inst.(fields)
	if ok {
		extraFields = fields.PrinterFields()
	}

	fmt.Fprint(wr, "NAME")
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[0])
	}
	fmt.Fprintln(wr)

	meta := phase.Metadata()
	fmt.Fprintf(wr, "%s", meta.Name)
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[1])
	}
	fmt.Fprintln(wr)

	return nil
}

type fields interface {
	PrinterFields() [][2]string
}

type labels map[string]string

func (l labels) String() string {
	return "KEY=VALUE"
}

func (l labels) Set(v string) error {
	key, value, match := strings.Cut(v, "=")
	if !match {
		return fmt.Errorf("value should be in the form key=value (found %q)", v)
	}

	l[key] = value

	return nil
}

func promote(ctx context.Context, s System, args ...string) error {
	var (
		labels = labels{}
		all    bool
		apply  bool
	)

	set := flag.NewFlagSet("promote", flag.ExitOnError)
	set.Var(&labels, "label", "selector for filtering phases (format key=value)")
	set.BoolVar(&apply, "apply", false, "actually run promotions (default dry-run)")
	set.BoolVar(&all, "all", false, "promote all phases (ignores label filters)")
	if err := set.Parse(args); err != nil {
		return err
	}

	var logArgs []any
	if !apply {
		logArgs = append(logArgs, "note", "use --apply for promotion to take effect (dry run)")
	}

	if all {
		// ignore labels if the all flag is passed
		labels = nil
	}

	for k, v := range labels {
		logArgs = append(logArgs, k, v)
	}

	slog.Info("starting promotion", logArgs...)

	if set.NArg() == 0 {
		if len(labels) == 0 && !all {
			return errors.New("please pass --all if you want to promote all phases")
		}

		for _, pipeline := range s.Pipelines() {
			if err := promoteAllPhases(ctx, pipeline.Phases(core.HasAllLabels(labels)), apply); err != nil {
				return err
			}
		}

		return nil
	}

	pipeline, err := s.GetPipeline(set.Arg(0))
	if err != nil {
		return err
	}

	phases := pipeline.Phases(core.HasAllLabels(labels))
	if set.NArg() < 2 {
		return promoteAllPhases(ctx, phases, apply)
	}

	phase, err := pipeline.PhaseByName(set.Arg(1))
	if err != nil {
		return err
	}

	if err := promoteAllPhases(ctx, toIter(phase), apply); err != nil {
		return err
	}

	return nil
}

func toIter[V any](v ...V) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, vv := range v {
			if !yield(vv) {
				break
			}
		}
	}
}

func promoteAllPhases(ctx context.Context, phases iter.Seq[core.Phase], apply bool) error {
	for phase := range phases {
		slog.Info("promoting phase", "phase", phase.Metadata().Name, "dry-run", !apply)

		if apply {
			if err := phase.Promote(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
