import { PhaseNode } from '@/types/flow';
import { Badge } from './ui/badge';
import { Package, GitBranch, ChevronDown, ChevronUp } from 'lucide-react';
import { getLabelColor } from '@/lib/utils';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';

interface NodePanelProps {
  node: PhaseNode | null;
  isExpanded: boolean;
  onToggle: () => void;
}

export function NodePanel({ node, isExpanded, onToggle }: NodePanelProps) {
  if (!node) return null;

  const getIcon = () => {
    switch (node.data.source.name ?? '') {
      case 'oci':
        return <Package className="h-4 w-4" />;
      default:
        return <GitBranch className="h-4 w-4" />;
    }
  };

  return (
    <div className="flex flex-col border-t bg-background">
      <div className="flex items-center justify-between px-4 py-2">
        <div className="flex items-center gap-2">
          {getIcon()}
          <h2 className="text-lg font-semibold">{node.data.name}</h2>
        </div>
        <Button variant="ghost" size="sm" onClick={onToggle}>
          {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
        </Button>
      </div>

      <div
        className={cn(
          'grid grid-cols-2 gap-4 overflow-hidden transition-[max-height,opacity] duration-300 ease-in-out',
          isExpanded ? 'max-h-[200px] opacity-100' : 'max-h-0 opacity-0'
        )}
      >
        <div className="space-y-4 overflow-hidden p-4">
          <div>
            <h3 className="text-sm font-medium">Details</h3>
            <div className="mt-2 space-y-2">
              <div className="text-sm">
                <span className="text-muted-foreground">Pipeline: </span>
                {node.data.pipeline}
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Depends on: </span>
                {node.data.depends_on || 'None'}
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Digest: </span>
                <span className="truncate font-mono">{node.data.digest}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-4 overflow-hidden p-4">
          <div>
            <h3 className="text-sm font-medium">Labels</h3>
            <div className="mt-2 flex flex-wrap gap-2">
              {node.data.labels &&
                Object.entries(node.data.labels).map(([key, value]) => (
                  <Badge
                    key={`${key}-${value}`}
                    className={`whitespace-nowrap text-xs font-light ${getLabelColor(key, value)}`}
                  >
                    {key}: {value}
                  </Badge>
                ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
