// Code generated from GluDSL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // GluDSL

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type GluDSLParser struct {
	*antlr.BaseParser
}

var GluDSLParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func gludslParserInit() {
	staticData := &GluDSLParserStaticData
	staticData.LiteralNames = []string{
		"", "'type'", "'name'", "'system'", "'pipeline'", "'source'", "'phase'",
		"'labels'", "'promotes_from'", "'{'", "'}'", "'='",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "SYSTEM", "PIPELINE", "SOURCE", "PHASE", "LABELS", "PROMOTES_FROM",
		"LBRACE", "RBRACE", "EQUALS", "STRING", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"program", "system", "systemBody", "pipeline", "pipelineBody", "source",
		"sourceBody", "phase", "phaseBody", "labels", "labelPair",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 14, 90, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 5, 2, 32, 8,
		2, 10, 2, 12, 2, 35, 9, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1,
		4, 5, 4, 45, 8, 4, 10, 4, 12, 4, 48, 9, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5,
		1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7,
		1, 8, 1, 8, 1, 8, 1, 8, 3, 8, 71, 8, 8, 1, 8, 3, 8, 74, 8, 8, 1, 9, 1,
		9, 1, 9, 5, 9, 79, 8, 9, 10, 9, 12, 9, 82, 9, 9, 1, 9, 1, 9, 1, 10, 1,
		10, 1, 10, 1, 10, 1, 10, 0, 0, 11, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20,
		0, 1, 1, 0, 12, 13, 84, 0, 22, 1, 0, 0, 0, 2, 24, 1, 0, 0, 0, 4, 33, 1,
		0, 0, 0, 6, 36, 1, 0, 0, 0, 8, 46, 1, 0, 0, 0, 10, 49, 1, 0, 0, 0, 12,
		55, 1, 0, 0, 0, 14, 60, 1, 0, 0, 0, 16, 66, 1, 0, 0, 0, 18, 75, 1, 0, 0,
		0, 20, 85, 1, 0, 0, 0, 22, 23, 3, 2, 1, 0, 23, 1, 1, 0, 0, 0, 24, 25, 5,
		3, 0, 0, 25, 26, 7, 0, 0, 0, 26, 27, 5, 9, 0, 0, 27, 28, 3, 4, 2, 0, 28,
		29, 5, 10, 0, 0, 29, 3, 1, 0, 0, 0, 30, 32, 3, 6, 3, 0, 31, 30, 1, 0, 0,
		0, 32, 35, 1, 0, 0, 0, 33, 31, 1, 0, 0, 0, 33, 34, 1, 0, 0, 0, 34, 5, 1,
		0, 0, 0, 35, 33, 1, 0, 0, 0, 36, 37, 5, 4, 0, 0, 37, 38, 7, 0, 0, 0, 38,
		39, 5, 9, 0, 0, 39, 40, 3, 8, 4, 0, 40, 41, 5, 10, 0, 0, 41, 7, 1, 0, 0,
		0, 42, 45, 3, 10, 5, 0, 43, 45, 3, 14, 7, 0, 44, 42, 1, 0, 0, 0, 44, 43,
		1, 0, 0, 0, 45, 48, 1, 0, 0, 0, 46, 44, 1, 0, 0, 0, 46, 47, 1, 0, 0, 0,
		47, 9, 1, 0, 0, 0, 48, 46, 1, 0, 0, 0, 49, 50, 5, 5, 0, 0, 50, 51, 7, 0,
		0, 0, 51, 52, 5, 9, 0, 0, 52, 53, 3, 12, 6, 0, 53, 54, 5, 10, 0, 0, 54,
		11, 1, 0, 0, 0, 55, 56, 5, 1, 0, 0, 56, 57, 7, 0, 0, 0, 57, 58, 5, 2, 0,
		0, 58, 59, 7, 0, 0, 0, 59, 13, 1, 0, 0, 0, 60, 61, 5, 6, 0, 0, 61, 62,
		7, 0, 0, 0, 62, 63, 5, 9, 0, 0, 63, 64, 3, 16, 8, 0, 64, 65, 5, 10, 0,
		0, 65, 15, 1, 0, 0, 0, 66, 67, 5, 5, 0, 0, 67, 70, 7, 0, 0, 0, 68, 69,
		5, 8, 0, 0, 69, 71, 7, 0, 0, 0, 70, 68, 1, 0, 0, 0, 70, 71, 1, 0, 0, 0,
		71, 73, 1, 0, 0, 0, 72, 74, 3, 18, 9, 0, 73, 72, 1, 0, 0, 0, 73, 74, 1,
		0, 0, 0, 74, 17, 1, 0, 0, 0, 75, 76, 5, 7, 0, 0, 76, 80, 5, 9, 0, 0, 77,
		79, 3, 20, 10, 0, 78, 77, 1, 0, 0, 0, 79, 82, 1, 0, 0, 0, 80, 78, 1, 0,
		0, 0, 80, 81, 1, 0, 0, 0, 81, 83, 1, 0, 0, 0, 82, 80, 1, 0, 0, 0, 83, 84,
		5, 10, 0, 0, 84, 19, 1, 0, 0, 0, 85, 86, 7, 0, 0, 0, 86, 87, 5, 11, 0,
		0, 87, 88, 5, 12, 0, 0, 88, 21, 1, 0, 0, 0, 6, 33, 44, 46, 70, 73, 80,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// GluDSLParserInit initializes any static state used to implement GluDSLParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewGluDSLParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func GluDSLParserInit() {
	staticData := &GluDSLParserStaticData
	staticData.once.Do(gludslParserInit)
}

// NewGluDSLParser produces a new parser instance for the optional input antlr.TokenStream.
func NewGluDSLParser(input antlr.TokenStream) *GluDSLParser {
	GluDSLParserInit()
	this := new(GluDSLParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &GluDSLParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "GluDSL.g4"

	return this
}

// GluDSLParser tokens.
const (
	GluDSLParserEOF           = antlr.TokenEOF
	GluDSLParserT__0          = 1
	GluDSLParserT__1          = 2
	GluDSLParserSYSTEM        = 3
	GluDSLParserPIPELINE      = 4
	GluDSLParserSOURCE        = 5
	GluDSLParserPHASE         = 6
	GluDSLParserLABELS        = 7
	GluDSLParserPROMOTES_FROM = 8
	GluDSLParserLBRACE        = 9
	GluDSLParserRBRACE        = 10
	GluDSLParserEQUALS        = 11
	GluDSLParserSTRING        = 12
	GluDSLParserIDENTIFIER    = 13
	GluDSLParserWS            = 14
)

// GluDSLParser rules.
const (
	GluDSLParserRULE_program      = 0
	GluDSLParserRULE_system       = 1
	GluDSLParserRULE_systemBody   = 2
	GluDSLParserRULE_pipeline     = 3
	GluDSLParserRULE_pipelineBody = 4
	GluDSLParserRULE_source       = 5
	GluDSLParserRULE_sourceBody   = 6
	GluDSLParserRULE_phase        = 7
	GluDSLParserRULE_phaseBody    = 8
	GluDSLParserRULE_labels       = 9
	GluDSLParserRULE_labelPair    = 10
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	System() ISystemContext

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) System() ISystemContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISystemContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISystemContext)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *GluDSLParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, GluDSLParserRULE_program)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(22)
		p.System()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISystemContext is an interface to support dynamic dispatch.
type ISystemContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SYSTEM() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	SystemBody() ISystemBodyContext
	RBRACE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsSystemContext differentiates from other interfaces.
	IsSystemContext()
}

type SystemContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySystemContext() *SystemContext {
	var p = new(SystemContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_system
	return p
}

func InitEmptySystemContext(p *SystemContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_system
}

func (*SystemContext) IsSystemContext() {}

func NewSystemContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SystemContext {
	var p = new(SystemContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_system

	return p
}

func (s *SystemContext) GetParser() antlr.Parser { return s.parser }

func (s *SystemContext) SYSTEM() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSYSTEM, 0)
}

func (s *SystemContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLBRACE, 0)
}

func (s *SystemContext) SystemBody() ISystemBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISystemBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISystemBodyContext)
}

func (s *SystemContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserRBRACE, 0)
}

func (s *SystemContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, 0)
}

func (s *SystemContext) STRING() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, 0)
}

func (s *SystemContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SystemContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SystemContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterSystem(s)
	}
}

func (s *SystemContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitSystem(s)
	}
}

func (p *GluDSLParser) System() (localctx ISystemContext) {
	localctx = NewSystemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, GluDSLParserRULE_system)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(24)
		p.Match(GluDSLParserSYSTEM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(25)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(26)
		p.Match(GluDSLParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(27)
		p.SystemBody()
	}
	{
		p.SetState(28)
		p.Match(GluDSLParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISystemBodyContext is an interface to support dynamic dispatch.
type ISystemBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllPipeline() []IPipelineContext
	Pipeline(i int) IPipelineContext

	// IsSystemBodyContext differentiates from other interfaces.
	IsSystemBodyContext()
}

type SystemBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySystemBodyContext() *SystemBodyContext {
	var p = new(SystemBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_systemBody
	return p
}

func InitEmptySystemBodyContext(p *SystemBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_systemBody
}

func (*SystemBodyContext) IsSystemBodyContext() {}

func NewSystemBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SystemBodyContext {
	var p = new(SystemBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_systemBody

	return p
}

func (s *SystemBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *SystemBodyContext) AllPipeline() []IPipelineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IPipelineContext); ok {
			len++
		}
	}

	tst := make([]IPipelineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IPipelineContext); ok {
			tst[i] = t.(IPipelineContext)
			i++
		}
	}

	return tst
}

func (s *SystemBodyContext) Pipeline(i int) IPipelineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPipelineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPipelineContext)
}

func (s *SystemBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SystemBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SystemBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterSystemBody(s)
	}
}

func (s *SystemBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitSystemBody(s)
	}
}

func (p *GluDSLParser) SystemBody() (localctx ISystemBodyContext) {
	localctx = NewSystemBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, GluDSLParserRULE_systemBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(33)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == GluDSLParserPIPELINE {
		{
			p.SetState(30)
			p.Pipeline()
		}

		p.SetState(35)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPipelineContext is an interface to support dynamic dispatch.
type IPipelineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PIPELINE() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	PipelineBody() IPipelineBodyContext
	RBRACE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsPipelineContext differentiates from other interfaces.
	IsPipelineContext()
}

type PipelineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPipelineContext() *PipelineContext {
	var p = new(PipelineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_pipeline
	return p
}

func InitEmptyPipelineContext(p *PipelineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_pipeline
}

func (*PipelineContext) IsPipelineContext() {}

func NewPipelineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PipelineContext {
	var p = new(PipelineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_pipeline

	return p
}

func (s *PipelineContext) GetParser() antlr.Parser { return s.parser }

func (s *PipelineContext) PIPELINE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserPIPELINE, 0)
}

func (s *PipelineContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLBRACE, 0)
}

func (s *PipelineContext) PipelineBody() IPipelineBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPipelineBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPipelineBodyContext)
}

func (s *PipelineContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserRBRACE, 0)
}

func (s *PipelineContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, 0)
}

func (s *PipelineContext) STRING() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, 0)
}

func (s *PipelineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PipelineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PipelineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterPipeline(s)
	}
}

func (s *PipelineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitPipeline(s)
	}
}

func (p *GluDSLParser) Pipeline() (localctx IPipelineContext) {
	localctx = NewPipelineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, GluDSLParserRULE_pipeline)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(36)
		p.Match(GluDSLParserPIPELINE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(37)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(38)
		p.Match(GluDSLParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(39)
		p.PipelineBody()
	}
	{
		p.SetState(40)
		p.Match(GluDSLParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPipelineBodyContext is an interface to support dynamic dispatch.
type IPipelineBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllSource() []ISourceContext
	Source(i int) ISourceContext
	AllPhase() []IPhaseContext
	Phase(i int) IPhaseContext

	// IsPipelineBodyContext differentiates from other interfaces.
	IsPipelineBodyContext()
}

type PipelineBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPipelineBodyContext() *PipelineBodyContext {
	var p = new(PipelineBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_pipelineBody
	return p
}

func InitEmptyPipelineBodyContext(p *PipelineBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_pipelineBody
}

func (*PipelineBodyContext) IsPipelineBodyContext() {}

func NewPipelineBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PipelineBodyContext {
	var p = new(PipelineBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_pipelineBody

	return p
}

func (s *PipelineBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *PipelineBodyContext) AllSource() []ISourceContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISourceContext); ok {
			len++
		}
	}

	tst := make([]ISourceContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISourceContext); ok {
			tst[i] = t.(ISourceContext)
			i++
		}
	}

	return tst
}

func (s *PipelineBodyContext) Source(i int) ISourceContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISourceContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISourceContext)
}

func (s *PipelineBodyContext) AllPhase() []IPhaseContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IPhaseContext); ok {
			len++
		}
	}

	tst := make([]IPhaseContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IPhaseContext); ok {
			tst[i] = t.(IPhaseContext)
			i++
		}
	}

	return tst
}

func (s *PipelineBodyContext) Phase(i int) IPhaseContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPhaseContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPhaseContext)
}

func (s *PipelineBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PipelineBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PipelineBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterPipelineBody(s)
	}
}

func (s *PipelineBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitPipelineBody(s)
	}
}

func (p *GluDSLParser) PipelineBody() (localctx IPipelineBodyContext) {
	localctx = NewPipelineBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, GluDSLParserRULE_pipelineBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(46)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == GluDSLParserSOURCE || _la == GluDSLParserPHASE {
		p.SetState(44)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case GluDSLParserSOURCE:
			{
				p.SetState(42)
				p.Source()
			}

		case GluDSLParserPHASE:
			{
				p.SetState(43)
				p.Phase()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(48)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISourceContext is an interface to support dynamic dispatch.
type ISourceContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SOURCE() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	SourceBody() ISourceBodyContext
	RBRACE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsSourceContext differentiates from other interfaces.
	IsSourceContext()
}

type SourceContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySourceContext() *SourceContext {
	var p = new(SourceContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_source
	return p
}

func InitEmptySourceContext(p *SourceContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_source
}

func (*SourceContext) IsSourceContext() {}

func NewSourceContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SourceContext {
	var p = new(SourceContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_source

	return p
}

func (s *SourceContext) GetParser() antlr.Parser { return s.parser }

func (s *SourceContext) SOURCE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSOURCE, 0)
}

func (s *SourceContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLBRACE, 0)
}

func (s *SourceContext) SourceBody() ISourceBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISourceBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISourceBodyContext)
}

func (s *SourceContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserRBRACE, 0)
}

func (s *SourceContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, 0)
}

func (s *SourceContext) STRING() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, 0)
}

func (s *SourceContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SourceContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SourceContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterSource(s)
	}
}

func (s *SourceContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitSource(s)
	}
}

func (p *GluDSLParser) Source() (localctx ISourceContext) {
	localctx = NewSourceContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, GluDSLParserRULE_source)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(49)
		p.Match(GluDSLParserSOURCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(50)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(51)
		p.Match(GluDSLParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(52)
		p.SourceBody()
	}
	{
		p.SetState(53)
		p.Match(GluDSLParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISourceBodyContext is an interface to support dynamic dispatch.
type ISourceBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	AllSTRING() []antlr.TerminalNode
	STRING(i int) antlr.TerminalNode

	// IsSourceBodyContext differentiates from other interfaces.
	IsSourceBodyContext()
}

type SourceBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySourceBodyContext() *SourceBodyContext {
	var p = new(SourceBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_sourceBody
	return p
}

func InitEmptySourceBodyContext(p *SourceBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_sourceBody
}

func (*SourceBodyContext) IsSourceBodyContext() {}

func NewSourceBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SourceBodyContext {
	var p = new(SourceBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_sourceBody

	return p
}

func (s *SourceBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *SourceBodyContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(GluDSLParserIDENTIFIER)
}

func (s *SourceBodyContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, i)
}

func (s *SourceBodyContext) AllSTRING() []antlr.TerminalNode {
	return s.GetTokens(GluDSLParserSTRING)
}

func (s *SourceBodyContext) STRING(i int) antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, i)
}

func (s *SourceBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SourceBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SourceBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterSourceBody(s)
	}
}

func (s *SourceBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitSourceBody(s)
	}
}

func (p *GluDSLParser) SourceBody() (localctx ISourceBodyContext) {
	localctx = NewSourceBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, GluDSLParserRULE_sourceBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(55)
		p.Match(GluDSLParserT__0)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(56)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(57)
		p.Match(GluDSLParserT__1)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(58)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPhaseContext is an interface to support dynamic dispatch.
type IPhaseContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PHASE() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	PhaseBody() IPhaseBodyContext
	RBRACE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	STRING() antlr.TerminalNode

	// IsPhaseContext differentiates from other interfaces.
	IsPhaseContext()
}

type PhaseContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPhaseContext() *PhaseContext {
	var p = new(PhaseContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_phase
	return p
}

func InitEmptyPhaseContext(p *PhaseContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_phase
}

func (*PhaseContext) IsPhaseContext() {}

func NewPhaseContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PhaseContext {
	var p = new(PhaseContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_phase

	return p
}

func (s *PhaseContext) GetParser() antlr.Parser { return s.parser }

func (s *PhaseContext) PHASE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserPHASE, 0)
}

func (s *PhaseContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLBRACE, 0)
}

func (s *PhaseContext) PhaseBody() IPhaseBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPhaseBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPhaseBodyContext)
}

func (s *PhaseContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserRBRACE, 0)
}

func (s *PhaseContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, 0)
}

func (s *PhaseContext) STRING() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, 0)
}

func (s *PhaseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PhaseContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PhaseContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterPhase(s)
	}
}

func (s *PhaseContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitPhase(s)
	}
}

func (p *GluDSLParser) Phase() (localctx IPhaseContext) {
	localctx = NewPhaseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, GluDSLParserRULE_phase)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(60)
		p.Match(GluDSLParserPHASE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(61)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(62)
		p.Match(GluDSLParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(63)
		p.PhaseBody()
	}
	{
		p.SetState(64)
		p.Match(GluDSLParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPhaseBodyContext is an interface to support dynamic dispatch.
type IPhaseBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SOURCE() antlr.TerminalNode
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	AllSTRING() []antlr.TerminalNode
	STRING(i int) antlr.TerminalNode
	PROMOTES_FROM() antlr.TerminalNode
	Labels() ILabelsContext

	// IsPhaseBodyContext differentiates from other interfaces.
	IsPhaseBodyContext()
}

type PhaseBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPhaseBodyContext() *PhaseBodyContext {
	var p = new(PhaseBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_phaseBody
	return p
}

func InitEmptyPhaseBodyContext(p *PhaseBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_phaseBody
}

func (*PhaseBodyContext) IsPhaseBodyContext() {}

func NewPhaseBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PhaseBodyContext {
	var p = new(PhaseBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_phaseBody

	return p
}

func (s *PhaseBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *PhaseBodyContext) SOURCE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserSOURCE, 0)
}

func (s *PhaseBodyContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(GluDSLParserIDENTIFIER)
}

func (s *PhaseBodyContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, i)
}

func (s *PhaseBodyContext) AllSTRING() []antlr.TerminalNode {
	return s.GetTokens(GluDSLParserSTRING)
}

func (s *PhaseBodyContext) STRING(i int) antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, i)
}

func (s *PhaseBodyContext) PROMOTES_FROM() antlr.TerminalNode {
	return s.GetToken(GluDSLParserPROMOTES_FROM, 0)
}

func (s *PhaseBodyContext) Labels() ILabelsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILabelsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILabelsContext)
}

func (s *PhaseBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PhaseBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PhaseBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterPhaseBody(s)
	}
}

func (s *PhaseBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitPhaseBody(s)
	}
}

func (p *GluDSLParser) PhaseBody() (localctx IPhaseBodyContext) {
	localctx = NewPhaseBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, GluDSLParserRULE_phaseBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(66)
		p.Match(GluDSLParserSOURCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(67)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(70)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == GluDSLParserPROMOTES_FROM {
		{
			p.SetState(68)
			p.Match(GluDSLParserPROMOTES_FROM)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(69)
			_la = p.GetTokenStream().LA(1)

			if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	p.SetState(73)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == GluDSLParserLABELS {
		{
			p.SetState(72)
			p.Labels()
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILabelsContext is an interface to support dynamic dispatch.
type ILabelsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LABELS() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllLabelPair() []ILabelPairContext
	LabelPair(i int) ILabelPairContext

	// IsLabelsContext differentiates from other interfaces.
	IsLabelsContext()
}

type LabelsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLabelsContext() *LabelsContext {
	var p = new(LabelsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_labels
	return p
}

func InitEmptyLabelsContext(p *LabelsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_labels
}

func (*LabelsContext) IsLabelsContext() {}

func NewLabelsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LabelsContext {
	var p = new(LabelsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_labels

	return p
}

func (s *LabelsContext) GetParser() antlr.Parser { return s.parser }

func (s *LabelsContext) LABELS() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLABELS, 0)
}

func (s *LabelsContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserLBRACE, 0)
}

func (s *LabelsContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(GluDSLParserRBRACE, 0)
}

func (s *LabelsContext) AllLabelPair() []ILabelPairContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILabelPairContext); ok {
			len++
		}
	}

	tst := make([]ILabelPairContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILabelPairContext); ok {
			tst[i] = t.(ILabelPairContext)
			i++
		}
	}

	return tst
}

func (s *LabelsContext) LabelPair(i int) ILabelPairContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILabelPairContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILabelPairContext)
}

func (s *LabelsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LabelsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LabelsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterLabels(s)
	}
}

func (s *LabelsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitLabels(s)
	}
}

func (p *GluDSLParser) Labels() (localctx ILabelsContext) {
	localctx = NewLabelsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, GluDSLParserRULE_labels)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(75)
		p.Match(GluDSLParserLABELS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(76)
		p.Match(GluDSLParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(80)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER {
		{
			p.SetState(77)
			p.LabelPair()
		}

		p.SetState(82)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(83)
		p.Match(GluDSLParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILabelPairContext is an interface to support dynamic dispatch.
type ILabelPairContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EQUALS() antlr.TerminalNode
	AllSTRING() []antlr.TerminalNode
	STRING(i int) antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode

	// IsLabelPairContext differentiates from other interfaces.
	IsLabelPairContext()
}

type LabelPairContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLabelPairContext() *LabelPairContext {
	var p = new(LabelPairContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_labelPair
	return p
}

func InitEmptyLabelPairContext(p *LabelPairContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = GluDSLParserRULE_labelPair
}

func (*LabelPairContext) IsLabelPairContext() {}

func NewLabelPairContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LabelPairContext {
	var p = new(LabelPairContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = GluDSLParserRULE_labelPair

	return p
}

func (s *LabelPairContext) GetParser() antlr.Parser { return s.parser }

func (s *LabelPairContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(GluDSLParserEQUALS, 0)
}

func (s *LabelPairContext) AllSTRING() []antlr.TerminalNode {
	return s.GetTokens(GluDSLParserSTRING)
}

func (s *LabelPairContext) STRING(i int) antlr.TerminalNode {
	return s.GetToken(GluDSLParserSTRING, i)
}

func (s *LabelPairContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(GluDSLParserIDENTIFIER, 0)
}

func (s *LabelPairContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LabelPairContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LabelPairContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.EnterLabelPair(s)
	}
}

func (s *LabelPairContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(GluDSLListener); ok {
		listenerT.ExitLabelPair(s)
	}
}

func (p *GluDSLParser) LabelPair() (localctx ILabelPairContext) {
	localctx = NewLabelPairContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, GluDSLParserRULE_labelPair)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(85)
		_la = p.GetTokenStream().LA(1)

		if !(_la == GluDSLParserSTRING || _la == GluDSLParserIDENTIFIER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	{
		p.SetState(86)
		p.Match(GluDSLParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(87)
		p.Match(GluDSLParserSTRING)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
