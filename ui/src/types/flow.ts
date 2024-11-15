import type { Edge, Node } from '@xyflow/react';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export type PipelineNode = PhaseNode | GroupNode;

type PhaseNodeData = {
  name: string;
  labels?: Record<string, string>;
};

export type PhaseNode = Node<PhaseNodeData, 'phase'>;

type GroupNodeData = {
  labels?: Record<string, string>;
};

export type GroupNode = Node<GroupNodeData, 'group'>;

export type PipelineEdge = Edge<{
  id: string;
  source: string;
  target: string;
}>;
