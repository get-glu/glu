import { PhaseNode } from '@/types/flow';
import {
  Package,
  GitBranch,
  ChevronDown,
  ChevronUp,
  CheckCircle,
  CircleArrowUp,
  CircleAlert
} from 'lucide-react';
import { TooltipProvider, TooltipTrigger, Tooltip, TooltipContent } from '@/components/ui/tooltip';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';
import { Label } from './label';

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
          {node.data.depends_on && node.data.depends_on !== '' && (
            <>
              {node.data.synced ? (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <CheckCircle className="h-4 w-4 flex-shrink-0 text-green-400" />
                    </TooltipTrigger>
                    <TooltipContent sideOffset={5} className="text-xs">
                      Up to Date
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <CircleAlert className="h-4 w-4 flex-shrink-0 cursor-pointer" />
                    </TooltipTrigger>
                    <TooltipContent sideOffset={5} className="text-xs">
                      Out of Sync
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </>
          )}
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
                <span className="truncate font-mono text-xs">{node.data.digest}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-4 overflow-hidden p-4">
          {node.data.labels && Object.keys(node.data.labels).length > 0 && (
            <div>
              <h3 className="text-sm font-medium">Labels</h3>
              <div className="mt-2 flex flex-wrap gap-2">
                {node.data.labels &&
                  Object.entries(node.data.labels).map(([key, value]) => (
                    <Label labelKey={key} value={value} />
                  ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
