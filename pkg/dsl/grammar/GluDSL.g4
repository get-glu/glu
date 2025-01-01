grammar GluDSL;

// Lexer Rules
SYSTEM: 'system';
PIPELINE: 'pipeline';
SOURCE: 'source';
PHASE: 'phase';
TRIGGER: 'trigger';
LABELS: 'labels';
PROMOTES_FROM: 'promotes_from';
INTERVAL: 'interval';
MATCHES_LABEL: 'matches_label';

LBRACE: '{';
RBRACE: '}';
EQUALS: '=';
STRING: '"' .*? '"';
IDENTIFIER: [a-zA-Z_][a-zA-Z0-9_-]*;

// Ignored tokens
WS: [ \t\r\n]+ -> skip;

// Parser Rules
program: system;

system: SYSTEM IDENTIFIER LBRACE systemBody RBRACE;

systemBody: (pipeline)*;

pipeline: PIPELINE IDENTIFIER LBRACE pipelineBody RBRACE;

pipelineBody: (source | phase | trigger)*;

source: SOURCE IDENTIFIER LBRACE sourceBody RBRACE;
sourceBody: 'type' IDENTIFIER 'name' IDENTIFIER;

phase: PHASE IDENTIFIER LBRACE phaseBody RBRACE;
phaseBody: 'source' IDENTIFIER (PROMOTES_FROM IDENTIFIER)? labels?;

labels: LABELS LBRACE (labelPair)* RBRACE;
labelPair: IDENTIFIER EQUALS STRING;

trigger: TRIGGER IDENTIFIER LBRACE triggerBody RBRACE;
triggerBody: INTERVAL STRING (MATCHES_LABEL IDENTIFIER ',' IDENTIFIER)?;