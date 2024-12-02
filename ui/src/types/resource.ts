import { Metadata } from './metadata';

export interface Resource {
  digest: string;
  metadata?: Metadata;
}
