import { PhaseNode } from '@/types/flow';
import { Package, GitBranch, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';
import { Label } from './label';
import { ScrollArea } from './ui/scroll-area';

interface PhasePanelProps {
  node: PhaseNode;
  isExpanded: boolean;
  onToggle: () => void;
}

export function PhasePanel({ node, isExpanded, onToggle }: PhasePanelProps) {
  const descriptor = node.data.descriptor;

  return (
    <div className="flex flex-col border-t bg-background">
      <div className="flex items-center justify-between border-b border-b-muted px-4 py-2">
        <div className="flex items-center gap-2">
          {node.data.kind === 'oci' ? (
            <Package className="h-4 w-4" />
          ) : (
            <GitBranch className="h-4 w-4" />
          )}
          <h2 className="text-lg font-semibold">{descriptor.metadata.name}</h2>
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
        <div className="overflow-hidden p-4">
          <h3 className="text-sm font-medium">Details</h3>
          <div className="mt-2 space-y-2">
            <div className="text-sm">
              <span className="text-muted-foreground">Pipeline: </span>
              {descriptor.pipeline}
            </div>
          </div>
        </div>

        <div className="h-[200px] p-4">
          {descriptor.metadata.labels && Object.keys(descriptor.metadata.labels).length > 0 && (
            <>
              <h3 className="text-sm font-medium">Labels</h3>
              <ScrollArea className="mt-2 h-[150px]">
                <div className="flex flex-wrap gap-2">
                  {Object.entries(descriptor.metadata.labels).map(([key, value]) => (
                    <Label key={key} labelKey={key} value={value} className="cursor-default" />
                  ))}
                </div>
              </ScrollArea>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
