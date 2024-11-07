import { Node } from '@xyflow/react';
import { PhaseData } from './pipeline';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export interface PipelineNode extends Node {
  type: 'phase' | 'group';
  data: PhaseData | { label: string };
  parentNode?: string;
}

export interface PipelineEdge {
  id: string;
  source: string;
  target: string;
}
