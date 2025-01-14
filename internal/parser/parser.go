package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/get-glu/glu/internal/core"
	"gopkg.in/yaml.v3"
)

type SystemConfig struct {
	Pipelines []PipelineConfig `yaml:"pipelines"`
}

type PipelineConfig struct {
	Name    string                  `yaml:"name"`
	Sources map[string]SourceConfig `yaml:"sources"`
	Phases  map[string]PhaseConfig  `yaml:"phases"`
}

type SourceConfig struct {
	Kind   string                 `yaml:"kind"`
	Name   string                 `yaml:"name"`
	SCM    string                 `yaml:"scm,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

type PhaseConfig struct {
	Source    string            `yaml:"source"`
	DependsOn string            `yaml:"depends_on,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func Parse(ctx context.Context, file string) (*core.System, error) {
	sys, err := readFromPath(ctx, os.DirFS("."), file)
	if err != nil {
		return nil, err
	}

	return sys, nil
}

// parsePipeline converts a PipelineConfig into a glu.Pipeline
func parsePipeline(_ context.Context, cfg PipelineConfig) (*core.Pipeline, error) {
	pipeline := core.NewPipeline(core.Name(cfg.Name))

	// Track phases for dependency linking
	phases := make(map[string]core.Phase)

	// First create all phases
	for phaseName, phaseConfig := range cfg.Phases {
		// Find the associated source
		sourceConfig, ok := cfg.Sources[phaseConfig.Source]
		if !ok {
			return nil, fmt.Errorf("phase %s references unknown source %s", phaseName, phaseConfig.Source)
		}

		// Create phase metadata
		phaseMeta := core.Name(phaseName)
		if phaseConfig.Labels != nil {
			for k, v := range phaseConfig.Labels {
				phaseMeta.Labels[k] = v
			}
		}

		// Create phase descriptor
		desc := core.Descriptor{
			Kind:     sourceConfig.Kind,
			Metadata: phaseMeta,
		}

		// Create the phase
		phase := core.NewPhase(desc)
		phases[phaseName] = phase
		pipeline.AddPhase(phase)
	}

	// Then link dependencies
	for phaseName, phaseConfig := range cfg.Phases {
		if phaseConfig.DependsOn != "" {
			currentPhase := phases[phaseName]
			dependentPhase, ok := phases[phaseConfig.DependsOn]
			if !ok {
				return nil, fmt.Errorf("phase %s depends on unknown phase %s", phaseName, phaseConfig.DependsOn)
			}

			pipeline.AddEdge(dependentPhase.Descriptor(), currentPhase.Descriptor())
		}
	}

	return pipeline, nil
}

func readFromPath(ctx context.Context, fs fs.FS, sysPath string) (_ *core.System, err error) {
	fi, err := fs.Open(sysPath)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	encoding := json.Unmarshal
	switch path.Ext(sysPath) {
	case ".yaml", ".yml":
		encoding = yaml.Unmarshal
	}

	return readFrom(ctx, &decoder{rd: fi, enc: encoding})
}

func readFrom(ctx context.Context, decoder *decoder) (_ *core.System, err error) {
	var sys SystemConfig
	if err := decoder.Decode(&sys); err != nil {
		return nil, err
	}

	system := core.NewSystem(ctx)
	for _, pipeline := range sys.Pipelines {
		pipeline, err := parsePipeline(ctx, pipeline)
		if err != nil {
			return nil, err
		}

		system.AddPipeline(pipeline)
	}

	return system, nil
}

type emptyReader struct{}

func (n emptyReader) Read(p []byte) (_ int, err error) {
	return 0, io.EOF
}

type encoding func([]byte, any) error

type decoder struct {
	rd  io.Reader
	enc encoding
}

func (d *decoder) Decode(v any) error {
	data, err := io.ReadAll(d.rd)
	if err != nil {
		return err
	}

	return d.enc(data, v)
}
