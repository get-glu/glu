import { Descriptor } from './descriptor';

// Server-side pipeline types
export interface Pipeline {
  name: string;
  labels: Record<string, string>;
  phases: Phase[];
  edges: Edge[];
}

export interface Phase {
  descriptor: Descriptor;
  status: Record<string, string>;
}

export interface Edge {
  kind: string;
  from: Descriptor;
  to: Descriptor;
}

export interface Result {
  annotations: Record<string, string>;
}
