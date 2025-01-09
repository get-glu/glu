// Code generated from GluDSL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type GluDSLLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var GluDSLLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func gludsllexerLexerInit() {
	staticData := &GluDSLLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'type'", "'name'", "'system'", "'pipeline'", "'source'", "'phase'",
		"'labels'", "'promotes_from'", "'{'", "'}'", "'='",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "SYSTEM", "PIPELINE", "SOURCE", "PHASE", "LABELS", "PROMOTES_FROM",
		"LBRACE", "RBRACE", "EQUALS", "STRING", "IDENTIFIER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "SYSTEM", "PIPELINE", "SOURCE", "PHASE", "LABELS", "PROMOTES_FROM",
		"LBRACE", "RBRACE", "EQUALS", "STRING", "IDENTIFIER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 14, 118, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 1, 0, 1, 0, 1, 0,
		1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2,
		1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4,
		1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5,
		1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7,
		1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9,
		1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 5, 11, 98, 8, 11, 10, 11, 12, 11, 101,
		9, 11, 1, 11, 1, 11, 1, 12, 1, 12, 5, 12, 107, 8, 12, 10, 12, 12, 12, 110,
		9, 12, 1, 13, 4, 13, 113, 8, 13, 11, 13, 12, 13, 114, 1, 13, 1, 13, 1,
		99, 0, 14, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19,
		10, 21, 11, 23, 12, 25, 13, 27, 14, 1, 0, 3, 3, 0, 65, 90, 95, 95, 97,
		122, 5, 0, 45, 45, 48, 57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13,
		32, 32, 120, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7,
		1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0, 0, 0, 0,
		15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1, 0, 0, 0,
		0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 1, 29, 1, 0, 0,
		0, 3, 34, 1, 0, 0, 0, 5, 39, 1, 0, 0, 0, 7, 46, 1, 0, 0, 0, 9, 55, 1, 0,
		0, 0, 11, 62, 1, 0, 0, 0, 13, 68, 1, 0, 0, 0, 15, 75, 1, 0, 0, 0, 17, 89,
		1, 0, 0, 0, 19, 91, 1, 0, 0, 0, 21, 93, 1, 0, 0, 0, 23, 95, 1, 0, 0, 0,
		25, 104, 1, 0, 0, 0, 27, 112, 1, 0, 0, 0, 29, 30, 5, 116, 0, 0, 30, 31,
		5, 121, 0, 0, 31, 32, 5, 112, 0, 0, 32, 33, 5, 101, 0, 0, 33, 2, 1, 0,
		0, 0, 34, 35, 5, 110, 0, 0, 35, 36, 5, 97, 0, 0, 36, 37, 5, 109, 0, 0,
		37, 38, 5, 101, 0, 0, 38, 4, 1, 0, 0, 0, 39, 40, 5, 115, 0, 0, 40, 41,
		5, 121, 0, 0, 41, 42, 5, 115, 0, 0, 42, 43, 5, 116, 0, 0, 43, 44, 5, 101,
		0, 0, 44, 45, 5, 109, 0, 0, 45, 6, 1, 0, 0, 0, 46, 47, 5, 112, 0, 0, 47,
		48, 5, 105, 0, 0, 48, 49, 5, 112, 0, 0, 49, 50, 5, 101, 0, 0, 50, 51, 5,
		108, 0, 0, 51, 52, 5, 105, 0, 0, 52, 53, 5, 110, 0, 0, 53, 54, 5, 101,
		0, 0, 54, 8, 1, 0, 0, 0, 55, 56, 5, 115, 0, 0, 56, 57, 5, 111, 0, 0, 57,
		58, 5, 117, 0, 0, 58, 59, 5, 114, 0, 0, 59, 60, 5, 99, 0, 0, 60, 61, 5,
		101, 0, 0, 61, 10, 1, 0, 0, 0, 62, 63, 5, 112, 0, 0, 63, 64, 5, 104, 0,
		0, 64, 65, 5, 97, 0, 0, 65, 66, 5, 115, 0, 0, 66, 67, 5, 101, 0, 0, 67,
		12, 1, 0, 0, 0, 68, 69, 5, 108, 0, 0, 69, 70, 5, 97, 0, 0, 70, 71, 5, 98,
		0, 0, 71, 72, 5, 101, 0, 0, 72, 73, 5, 108, 0, 0, 73, 74, 5, 115, 0, 0,
		74, 14, 1, 0, 0, 0, 75, 76, 5, 112, 0, 0, 76, 77, 5, 114, 0, 0, 77, 78,
		5, 111, 0, 0, 78, 79, 5, 109, 0, 0, 79, 80, 5, 111, 0, 0, 80, 81, 5, 116,
		0, 0, 81, 82, 5, 101, 0, 0, 82, 83, 5, 115, 0, 0, 83, 84, 5, 95, 0, 0,
		84, 85, 5, 102, 0, 0, 85, 86, 5, 114, 0, 0, 86, 87, 5, 111, 0, 0, 87, 88,
		5, 109, 0, 0, 88, 16, 1, 0, 0, 0, 89, 90, 5, 123, 0, 0, 90, 18, 1, 0, 0,
		0, 91, 92, 5, 125, 0, 0, 92, 20, 1, 0, 0, 0, 93, 94, 5, 61, 0, 0, 94, 22,
		1, 0, 0, 0, 95, 99, 5, 34, 0, 0, 96, 98, 9, 0, 0, 0, 97, 96, 1, 0, 0, 0,
		98, 101, 1, 0, 0, 0, 99, 100, 1, 0, 0, 0, 99, 97, 1, 0, 0, 0, 100, 102,
		1, 0, 0, 0, 101, 99, 1, 0, 0, 0, 102, 103, 5, 34, 0, 0, 103, 24, 1, 0,
		0, 0, 104, 108, 7, 0, 0, 0, 105, 107, 7, 1, 0, 0, 106, 105, 1, 0, 0, 0,
		107, 110, 1, 0, 0, 0, 108, 106, 1, 0, 0, 0, 108, 109, 1, 0, 0, 0, 109,
		26, 1, 0, 0, 0, 110, 108, 1, 0, 0, 0, 111, 113, 7, 2, 0, 0, 112, 111, 1,
		0, 0, 0, 113, 114, 1, 0, 0, 0, 114, 112, 1, 0, 0, 0, 114, 115, 1, 0, 0,
		0, 115, 116, 1, 0, 0, 0, 116, 117, 6, 13, 0, 0, 117, 28, 1, 0, 0, 0, 4,
		0, 99, 108, 114, 1, 6, 0, 0,
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

// GluDSLLexerInit initializes any static state used to implement GluDSLLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewGluDSLLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func GluDSLLexerInit() {
	staticData := &GluDSLLexerLexerStaticData
	staticData.once.Do(gludsllexerLexerInit)
}

// NewGluDSLLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewGluDSLLexer(input antlr.CharStream) *GluDSLLexer {
	GluDSLLexerInit()
	l := new(GluDSLLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &GluDSLLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "GluDSL.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// GluDSLLexer tokens.
const (
	GluDSLLexerT__0          = 1
	GluDSLLexerT__1          = 2
	GluDSLLexerSYSTEM        = 3
	GluDSLLexerPIPELINE      = 4
	GluDSLLexerSOURCE        = 5
	GluDSLLexerPHASE         = 6
	GluDSLLexerLABELS        = 7
	GluDSLLexerPROMOTES_FROM = 8
	GluDSLLexerLBRACE        = 9
	GluDSLLexerRBRACE        = 10
	GluDSLLexerEQUALS        = 11
	GluDSLLexerSTRING        = 12
	GluDSLLexerIDENTIFIER    = 13
	GluDSLLexerWS            = 14
)
