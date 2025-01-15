import { Metadata } from './metadata';
import { Package, GitBranch, Container, Hexagon, Github, Gitlab } from 'lucide-react';
export type Descriptor = {
  pipeline: string;
  metadata: Metadata;
  source: Source;
};

export type Source = {
  kind: string;
  name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config: Record<string, any>;
};
