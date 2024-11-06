import { Node } from '@xyflow/react';
import { PhaseData, ResourceData } from './pipeline';

export interface FlowPipeline {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export interface PipelineNode extends Node {
  type: 'phase' | 'resource';
  data: PhaseData | ResourceData;
}

export interface PipelineEdge {
  id: string;
  source: string;
  target: string;
}
