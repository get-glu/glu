grammar GluDSL;

// Lexer Rules
SYSTEM: 'system';
PIPELINE: 'pipeline';
SOURCE: 'source';
PHASE: 'phase';
LABELS: 'labels';
PROMOTES_FROM: 'promotes_from';

LBRACE: '{';
RBRACE: '}';
EQUALS: '=';
STRING: '"' .*? '"';
IDENTIFIER: [a-zA-Z_][a-zA-Z0-9_-]*;

// Ignored tokens
WS: [ \t\r\n]+ -> skip;

// Parser Rules
program: system;

system: SYSTEM (IDENTIFIER | STRING) LBRACE systemBody RBRACE;

systemBody: (pipeline)*;

pipeline: PIPELINE (IDENTIFIER | STRING) LBRACE pipelineBody RBRACE;

pipelineBody: (source | phase )*;

source: SOURCE (IDENTIFIER | STRING) LBRACE sourceBody RBRACE;
sourceBody: 'type' (IDENTIFIER | STRING) 'name' (IDENTIFIER | STRING);

phase: PHASE (IDENTIFIER | STRING) LBRACE phaseBody RBRACE;
phaseBody: 'source' (IDENTIFIER | STRING) (PROMOTES_FROM (IDENTIFIER | STRING))? labels?;

labels: LABELS LBRACE (labelPair)* RBRACE;
labelPair: (IDENTIFIER | STRING) EQUALS STRING;