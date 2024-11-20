import type { Edge, Node } from '@xyflow/react';
import { Metadata } from './metadata';

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
  source: Metadata;
  digest?: string;
};

export type PhaseNode = Node<PhaseNodeData, 'phase'>;

export type PipelineEdge = Edge<{
  id: string;
  source: string;
  target: string;
}>;
