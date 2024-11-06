// Server-side pipeline types
export interface Pipeline {
  id: string;
  name: string;
  phases: Phase[];
}

export interface Phase {
  id: string;
  name: string;
  type: 'oci' | 'staging' | 'production';
  dependsOn?: string;
  pipelineId: string;
}

export interface PhaseData extends Record<string, unknown> {
  label: string;
  type: 'oci' | 'staging' | 'production';
}

export interface PipelineGroup {
  id: string;
  name: string;
}
