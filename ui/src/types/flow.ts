import type { Edge, Node } from '@xyflow/react';
import { Phase } from './pipeline';
import { Descriptor } from './descriptor';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export type PipelineNode = PhaseNode;

type PhaseNodeData = Phase & Record<string, unknown>;

export type PhaseNode = Node<PhaseNodeData, 'phase'>;

export type PipelineEdge = Edge<{
  kind: string;
  from: Descriptor;
  to: Descriptor;
  can_perform: boolean;
}>;
