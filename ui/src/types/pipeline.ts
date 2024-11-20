// Server-side pipeline types
export interface Pipeline {
  name: string;
  phases: Phase[];
}

export interface Phase {
  name: string;
  depends_on?: string;
  source_type?: string;
  digest?: string;
  labels?: Record<string, string>;
  value?: Record<string, unknown>;
}

export interface PipelineGroup {
  id: string;
  name: string;
}
