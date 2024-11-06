// Server-side pipeline types
export interface Pipeline {
  name: string;
  phases: Phase[];
}

export interface Phase {
  name: string;
  resources: string[];
}

export interface PhaseData extends Record<string, unknown> {
  label: string;
  resources: string[];
  environment: string;
  color: string;
}

export interface ResourceData extends Record<string, unknown> {
  label: string;
  type: 'git' | 'artifact' | 'image'; // Add other resource types as needed
}
