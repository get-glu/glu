import type { Edge, Node } from '@xyflow/react';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export type PipelineNode = PhaseNode;

type PhaseNodeData = {
  pipeline: string;
  name: string;
  labels?: Record<string, string>;
  depends_on?: string;
  source_type?: string;
  digest?: string;
  value?: Record<string, unknown>;
};

export type PhaseNode = Node<PhaseNodeData, 'phase'>;

export type PipelineEdge = Edge<{
  id: string;
  source: string;
  target: string;
}>;
