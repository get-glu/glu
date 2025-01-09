// Code generated from GluDSL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // GluDSL

import "github.com/antlr4-go/antlr/v4"

// BaseGluDSLListener is a complete listener for a parse tree produced by GluDSLParser.
type BaseGluDSLListener struct{}

var _ GluDSLListener = &BaseGluDSLListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseGluDSLListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseGluDSLListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseGluDSLListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseGluDSLListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProgram is called when production program is entered.
func (s *BaseGluDSLListener) EnterProgram(ctx *ProgramContext) {}

// ExitProgram is called when production program is exited.
func (s *BaseGluDSLListener) ExitProgram(ctx *ProgramContext) {}

// EnterSystem is called when production system is entered.
func (s *BaseGluDSLListener) EnterSystem(ctx *SystemContext) {}

// ExitSystem is called when production system is exited.
func (s *BaseGluDSLListener) ExitSystem(ctx *SystemContext) {}

// EnterSystemBody is called when production systemBody is entered.
func (s *BaseGluDSLListener) EnterSystemBody(ctx *SystemBodyContext) {}

// ExitSystemBody is called when production systemBody is exited.
func (s *BaseGluDSLListener) ExitSystemBody(ctx *SystemBodyContext) {}

// EnterPipeline is called when production pipeline is entered.
func (s *BaseGluDSLListener) EnterPipeline(ctx *PipelineContext) {}

// ExitPipeline is called when production pipeline is exited.
func (s *BaseGluDSLListener) ExitPipeline(ctx *PipelineContext) {}

// EnterPipelineBody is called when production pipelineBody is entered.
func (s *BaseGluDSLListener) EnterPipelineBody(ctx *PipelineBodyContext) {}

// ExitPipelineBody is called when production pipelineBody is exited.
func (s *BaseGluDSLListener) ExitPipelineBody(ctx *PipelineBodyContext) {}

// EnterSource is called when production source is entered.
func (s *BaseGluDSLListener) EnterSource(ctx *SourceContext) {}

// ExitSource is called when production source is exited.
func (s *BaseGluDSLListener) ExitSource(ctx *SourceContext) {}

// EnterSourceBody is called when production sourceBody is entered.
func (s *BaseGluDSLListener) EnterSourceBody(ctx *SourceBodyContext) {}

// ExitSourceBody is called when production sourceBody is exited.
func (s *BaseGluDSLListener) ExitSourceBody(ctx *SourceBodyContext) {}

// EnterPhase is called when production phase is entered.
func (s *BaseGluDSLListener) EnterPhase(ctx *PhaseContext) {}

// ExitPhase is called when production phase is exited.
func (s *BaseGluDSLListener) ExitPhase(ctx *PhaseContext) {}

// EnterPhaseBody is called when production phaseBody is entered.
func (s *BaseGluDSLListener) EnterPhaseBody(ctx *PhaseBodyContext) {}

// ExitPhaseBody is called when production phaseBody is exited.
func (s *BaseGluDSLListener) ExitPhaseBody(ctx *PhaseBodyContext) {}

// EnterLabels is called when production labels is entered.
func (s *BaseGluDSLListener) EnterLabels(ctx *LabelsContext) {}

// ExitLabels is called when production labels is exited.
func (s *BaseGluDSLListener) ExitLabels(ctx *LabelsContext) {}

// EnterLabelPair is called when production labelPair is entered.
func (s *BaseGluDSLListener) EnterLabelPair(ctx *LabelPairContext) {}

// ExitLabelPair is called when production labelPair is exited.
func (s *BaseGluDSLListener) ExitLabelPair(ctx *LabelPairContext) {}
