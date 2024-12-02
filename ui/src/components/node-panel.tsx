import { PipelineNode } from '@/types/flow';
import { Package, GitBranch, ChevronDown, ChevronUp, History } from 'lucide-react';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';
import { Label } from './label';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from './ui/sheet';
import { useState } from 'react';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './ui/tooltip';

// Add mock history data
const MOCK_HISTORY = Array.from({ length: 10 }, (_, i) => ({
  version: `sha256:${Math.random().toString(36).substring(2, 15)}${Math.random().toString(36).substring(2, 15)}`
}));

interface NodePanelProps {
  node: PipelineNode;
  isExpanded: boolean;
  onToggle: () => void;
}

export function NodePanel({ node, isExpanded, onToggle }: NodePanelProps) {
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);

  // TODO: Replace with API query
  const history = MOCK_HISTORY;

  return (
    <div className="flex flex-col border-t bg-background">
      <div className="flex items-center justify-between px-4 pt-2">
        <div className="flex items-center gap-2">
          {node.data.kind === 'oci' ? (
            <Package className="h-4 w-4" />
          ) : (
            <GitBranch className="h-4 w-4" />
          )}
          <h2 className="text-lg font-semibold">{node.data.descriptor.metadata.name}</h2>
          <Sheet open={isHistorySheetOpen} onOpenChange={setIsHistorySheetOpen}>
            <SheetTrigger>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <History className="h-4 w-4 transition-transform hover:-rotate-45 hover:scale-125" />
                  </TooltipTrigger>
                  <TooltipContent sideOffset={5} className="text-sm">
                    Show History
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </SheetTrigger>
            <SheetContent>
              <SheetHeader>
                <SheetTitle>Phase History</SheetTitle>
              </SheetHeader>
              <div className="mt-6 max-h-[calc(100vh-8rem)] space-y-2 overflow-y-auto">
                {history.map((state) => (
                  <div key={state.version} className="text-sm">
                    <span className="text-muted-foreground">Version: </span>
                    <span className="font-mono text-xs">{state.version}</span>
                  </div>
                ))}
              </div>
            </SheetContent>
          </Sheet>
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
          <div>
            <h3 className="text-sm font-medium">Details</h3>
            <div className="mt-2 space-y-2">
              <div className="text-sm">
                <span className="text-muted-foreground">Pipeline: </span>
                {node.data.descriptor.pipeline}
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Digest: </span>
                <span className="truncate font-mono text-xs">{node.data.resource.digest}</span>
              </div>
            </div>
          </div>
        </div>

        <div className="overflow-hidden p-4">
          {node.data.descriptor.metadata.labels &&
            Object.keys(node.data.descriptor.metadata.labels).length > 0 && (
              <div>
                <h3 className="text-sm font-medium">Labels</h3>
                <div className="mt-2 flex flex-wrap gap-2">
                  {Object.entries(node.data.descriptor.metadata.labels).map(([key, value]) => (
                    <Label key={key} labelKey={key} value={value} />
                  ))}
                </div>
              </div>
            )}
        </div>
      </div>
    </div>
  );
}
