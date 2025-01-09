package parser

import (
	"github.com/antlr4-go/antlr/v4"
)

// Pipeline represents the parsed pipeline structure
type Pipeline struct {
	Name    string
	Sources map[string]*Source
	Phases  map[string]*Phase
}

type Source struct {
	Type string
	Name string
}

type Phase struct {
	SourceRef    string
	PromotesFrom string
	Labels       map[string]string
}

// Helper function to get text without quotes
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// GluListener implements the generated GluDSLListener interface
type GluListener struct {
	*BaseGluDSLListener
	CurrentSystem   string
	CurrentPipeline *Pipeline
	CurrentSource   *Source
	CurrentPhase    *Phase
	Systems         map[string][]*Pipeline
}

func (l *GluListener) EnterSystem(ctx *SystemContext) {
	if ctx.IDENTIFIER() != nil {
		l.CurrentSystem = ctx.IDENTIFIER().GetText()
	} else if ctx.STRING() != nil {
		l.CurrentSystem = unquote(ctx.STRING().GetText())
	}
	l.Systems[l.CurrentSystem] = make([]*Pipeline, 0)
}

func (l *GluListener) EnterPipeline(ctx *PipelineContext) {
	var name string
	if ctx.IDENTIFIER() != nil {
		name = ctx.IDENTIFIER().GetText()
	} else if ctx.STRING() != nil {
		name = unquote(ctx.STRING().GetText())
	}

	l.CurrentPipeline = &Pipeline{
		Name:    name,
		Sources: make(map[string]*Source),
		Phases:  make(map[string]*Phase),
	}
}

func (l *GluListener) ExitPipeline(ctx *PipelineContext) {
	l.Systems[l.CurrentSystem] = append(l.Systems[l.CurrentSystem], l.CurrentPipeline)
}

func (l *GluListener) EnterSource(ctx *SourceContext) {
	l.CurrentSource = &Source{}
}

func (l *GluListener) ExitSourceBody(ctx *SourceBodyContext) {
	// Get type and name from the source body
	typeToken := ctx.GetChild(1).(antlr.TerminalNode)
	nameToken := ctx.GetChild(3).(antlr.TerminalNode)

	l.CurrentSource.Type = unquote(typeToken.GetText())
	l.CurrentSource.Name = unquote(nameToken.GetText())

	// Get the source name from parent context
	sourceCtx := ctx.GetParent().(*SourceContext)
	var sourceName string
	if sourceCtx.IDENTIFIER() != nil {
		sourceName = sourceCtx.IDENTIFIER().GetText()
	} else if sourceCtx.STRING() != nil {
		sourceName = unquote(sourceCtx.STRING().GetText())
	}

	l.CurrentPipeline.Sources[sourceName] = l.CurrentSource
}

func (l *GluListener) EnterPhase(ctx *PhaseContext) {
	l.CurrentPhase = &Phase{
		Labels: make(map[string]string),
	}
}

func (l *GluListener) ExitPhaseBody(ctx *PhaseBodyContext) {
	sourceToken := ctx.GetChild(1).(antlr.TerminalNode)
	l.CurrentPhase.SourceRef = unquote(sourceToken.GetText())

	if ctx.PROMOTES_FROM() != nil {
		promotesToken := ctx.GetChild(3).(antlr.TerminalNode)
		l.CurrentPhase.PromotesFrom = unquote(promotesToken.GetText())
	}

	phaseCtx := ctx.GetParent().(*PhaseContext)
	var phaseName string
	if phaseCtx.IDENTIFIER() != nil {
		phaseName = phaseCtx.IDENTIFIER().GetText()
	} else if phaseCtx.STRING() != nil {
		phaseName = unquote(phaseCtx.STRING().GetText())
	}

	l.CurrentPipeline.Phases[phaseName] = l.CurrentPhase
}

func ParsePipeline(input string) (map[string][]*Pipeline, error) {
	inputStream := antlr.NewInputStream(input)
	lexer := NewGluDSLLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := NewGluDSLParser(stream)

	listener := &GluListener{
		Systems: make(map[string][]*Pipeline),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parser.Program())

	return listener.Systems, nil
}
