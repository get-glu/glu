import { Node } from '@xyflow/react';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export interface PipelineNode extends Node {
  type: 'phase' | 'group';
  data: any;
  parentNode?: string;
}

export interface PipelineEdge {
  id: string;
  source: string;
  target: string;
}
