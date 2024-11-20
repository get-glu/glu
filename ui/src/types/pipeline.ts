// Server-side pipeline types
export interface Pipeline {
  name: string;
  phases: Phase[];
}

type Resource = {
  digest: string;
  [key: string]: unknown;
};

export interface Phase {
  name: string;
  depends_on?: string;
  source_type?: string;
  digest?: string;
  labels?: Record<string, string>;
  value?: Resource;
}

export interface PipelineGroup {
  id: string;
  name: string;
}
