import { type LucideIcon } from 'lucide-react';
import { Package, GitBranch, Container, Hexagon, Github, Gitlab } from 'lucide-react';
import { Source } from './descriptor';

export type Icon = LucideIcon;

export const getSourceIcon = (source: Source): Icon => {
  switch (source.kind ?? '') {
    case 'kubernetes':
    case 'k8s':
      return Container;
    case 'oci':
      return Package;
    case 'ci':
      if (source.config?.scm === 'github') {
        return Github;
      } else if (source.config?.scm === 'gitlab') {
        return Gitlab;
      }
      return GitBranch;
    case 'git':
      return GitBranch;
    default:
      return Hexagon;
  }
};
