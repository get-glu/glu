package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/get-glu/glu/pkg/core"
)

type System interface {
	GetPipeline(name string) (*core.Pipeline, error)
	Pipelines() iter.Seq2[string, *core.Pipeline]
}

func Run(ctx context.Context, s System, args ...string) error {
	switch args[1] {
	case "inspect":
		return inspect(ctx, s, args[2:]...)
	case "do":
		return do(ctx, s, args[2:]...)
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
		for _, row := range pipelineRows(pipeline) {
			fmt.Fprintln(wr, strings.Join(row, "\t"))
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

	fmt.Fprint(wr, "NAME\tDIGEST")
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[0])
	}
	fmt.Fprintln(wr)

	meta := phase.Descriptor().Metadata
	digest, err := inst.Digest()
	if err != nil {
		return err
	}

	fmt.Fprintf(wr, "%s\t%s", meta.Name, digest)
	for _, field := range extraFields {
		fmt.Fprintf(wr, "\t%s", field[1])
	}
	fmt.Fprintln(wr)

	return nil
}

type fields interface {
	PrinterFields() [][2]string
}

func do(ctx context.Context, s System, args ...string) error {
	var (
		apply bool
		from  string
		to    string
	)

	set := flag.NewFlagSet("promote", flag.ExitOnError)
	set.BoolVar(&apply, "apply", false, "actually run promotions (default dry-run)")
	set.StringVar(&from, "from", "", "source phase name to perform from")
	set.StringVar(&to, "to", "", "destination phase name to perform to")
	if err := set.Parse(args); err != nil {
		return err
	}

	var logArgs []any
	if !apply {
		logArgs = append(logArgs, "note", "use --apply for promotion to take effect (dry run)")
	}

	if set.NArg() < 2 {
		return errors.New("glu do [pipeline] [kind] <[direction]=[name]>")
	}

	pipeline, err := s.GetPipeline(set.Arg(0))
	if err != nil {
		return err
	}

	kind := set.Arg(1)

	logArgs = append(logArgs, "pipeline", set.Arg(0), "kind", kind)

	for edge := range pipeline.Edges(core.WithKind(kind)) {
		if from != "" && edge.From().Metadata.Name != from {
			continue
		}

		if to != "" && edge.To().Metadata.Name != to {
			continue
		}

		slog.Info("performing", append(logArgs, "from", edge.From().Metadata.Name, "to", edge.To().Metadata.Name)...)
		if apply {
			if _, err := edge.Perform(ctx); err != nil {
				return err
			}
		}
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

func pipelineRows(pipeline *core.Pipeline) (phases [][]string) {
	var edges []string

	for _, out := range pipeline.EdgesFrom() {
		for _, edge := range out {
			kind := strings.ToUpper(edge.Kind())
			if idx, match := slices.BinarySearch(edges, kind); !match {
				edges = slices.Insert(edges, idx, kind)
			}
		}
	}

	for _, phase := range slices.SortedFunc(pipeline.Phases(), func(a, b core.Phase) int {
		return strings.Compare(a.Descriptor().Metadata.Name,
			b.Descriptor().Metadata.Name)
	}) {
		p := make([]string, len(edges)+1)
		p[0] = phase.Descriptor().Metadata.Name
		for _, edge := range pipeline.EdgesFrom()[phase.Descriptor().Metadata.Name] {
			idx, _ := slices.BinarySearch(edges, strings.ToUpper(edge.Kind()))
			p[idx+1] = edge.To().Metadata.Name
		}

		phases = append(phases, p)
	}

	return append([][]string{append([]string{"NAME"}, edges...)}, phases...)
}
