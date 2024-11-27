import { Metadata } from './metadata';
import { Resource } from './resource';

// Server-side pipeline types
export interface Pipeline {
  name: string;
  phases: Phase[];
}

export interface Phase {
  metadata: Metadata;
  source: Metadata;
  depends_on?: string;
  resource: Resource;
}
