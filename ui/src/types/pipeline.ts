import { Descriptor } from './descriptor';
import { Resource } from './resource';

// Server-side pipeline types
export interface Pipeline {
  name: string;
  labels: Record<string, string>;
  phases: Phase[];
  edges: Edge[];
}

export interface Phase {
  descriptor: Descriptor;
  resource: Resource;
}

export interface Edge {
  kind: string;
  from: Descriptor;
  to: Descriptor;
  can_perform?: boolean;
}

export interface Result {
  annotations: Record<string, string>;
}

export interface State {
  version: string;
  resource: Resource;
  annotations: Record<string, string>;
}
