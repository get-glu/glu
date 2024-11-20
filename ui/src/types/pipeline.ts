import { Metadata } from './metadata';

// Server-side pipeline types
export interface Pipeline {
  name: string;
  phases: Phase[];
}

export interface Phase {
  name: string;
  depends_on?: string;
  source: Metadata;
  digest?: string;
  labels?: Record<string, string>;
}

export interface PipelineGroup {
  id: string;
  name: string;
}
