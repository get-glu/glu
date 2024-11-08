// Server-side pipeline types
export interface Pipeline {
  name: string;
  controllers: Controller[];
}

export interface Controller {
  name: string;
  depends_on?: string;
  labels?: Record<string, string>;
  value?: unknown;
}

export interface PipelineGroup {
  id: string;
  name: string;
}
