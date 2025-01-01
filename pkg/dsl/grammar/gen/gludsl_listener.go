// Code generated from GluDSL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // GluDSL

import "github.com/antlr4-go/antlr/v4"

// GluDSLListener is a complete listener for a parse tree produced by GluDSLParser.
type GluDSLListener interface {
	antlr.ParseTreeListener

	// EnterProgram is called when entering the program production.
	EnterProgram(c *ProgramContext)

	// EnterSystem is called when entering the system production.
	EnterSystem(c *SystemContext)

	// EnterSystemBody is called when entering the systemBody production.
	EnterSystemBody(c *SystemBodyContext)

	// EnterPipeline is called when entering the pipeline production.
	EnterPipeline(c *PipelineContext)

	// EnterPipelineBody is called when entering the pipelineBody production.
	EnterPipelineBody(c *PipelineBodyContext)

	// EnterSource is called when entering the source production.
	EnterSource(c *SourceContext)

	// EnterSourceBody is called when entering the sourceBody production.
	EnterSourceBody(c *SourceBodyContext)

	// EnterPhase is called when entering the phase production.
	EnterPhase(c *PhaseContext)

	// EnterPhaseBody is called when entering the phaseBody production.
	EnterPhaseBody(c *PhaseBodyContext)

	// EnterLabels is called when entering the labels production.
	EnterLabels(c *LabelsContext)

	// EnterLabelPair is called when entering the labelPair production.
	EnterLabelPair(c *LabelPairContext)

	// EnterTrigger is called when entering the trigger production.
	EnterTrigger(c *TriggerContext)

	// EnterTriggerBody is called when entering the triggerBody production.
	EnterTriggerBody(c *TriggerBodyContext)

	// ExitProgram is called when exiting the program production.
	ExitProgram(c *ProgramContext)

	// ExitSystem is called when exiting the system production.
	ExitSystem(c *SystemContext)

	// ExitSystemBody is called when exiting the systemBody production.
	ExitSystemBody(c *SystemBodyContext)

	// ExitPipeline is called when exiting the pipeline production.
	ExitPipeline(c *PipelineContext)

	// ExitPipelineBody is called when exiting the pipelineBody production.
	ExitPipelineBody(c *PipelineBodyContext)

	// ExitSource is called when exiting the source production.
	ExitSource(c *SourceContext)

	// ExitSourceBody is called when exiting the sourceBody production.
	ExitSourceBody(c *SourceBodyContext)

	// ExitPhase is called when exiting the phase production.
	ExitPhase(c *PhaseContext)

	// ExitPhaseBody is called when exiting the phaseBody production.
	ExitPhaseBody(c *PhaseBodyContext)

	// ExitLabels is called when exiting the labels production.
	ExitLabels(c *LabelsContext)

	// ExitLabelPair is called when exiting the labelPair production.
	ExitLabelPair(c *LabelPairContext)

	// ExitTrigger is called when exiting the trigger production.
	ExitTrigger(c *TriggerContext)

	// ExitTriggerBody is called when exiting the triggerBody production.
	ExitTriggerBody(c *TriggerBodyContext)
}
