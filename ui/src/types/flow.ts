import type { Edge, Node } from '@xyflow/react';
import { Metadata } from './metadata';
import { Phase } from './pipeline';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export type PipelineNode = PhaseNode;

type PhaseNodeData = Phase &
  Record<string, unknown> & {
    pipeline: string;
  };

export type PhaseNode = Node<PhaseNodeData, 'phase'>;

export type PipelineEdge = Edge<{
  id: string;
  source: string;
  target: string;
}>;
