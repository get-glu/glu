import { formatDistanceToNow } from 'date-fns';
import { TooltipProvider, Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger
} from '@/components/ui/sheet';
import { useState } from 'react';
import { History } from 'lucide-react';
import { State } from '@/types/pipeline';

export function PhaseHistory({ phase, history }: { phase: string; history: State[] }) {
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);

  return (
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
          <SheetTitle>{phase}</SheetTitle>
          <SheetDescription>Phase History</SheetDescription>
        </SheetHeader>
        <div className="mt-6 max-h-[calc(100vh-8rem)] space-y-6">
          {history.map((state, index) => (
            <div key={index} className="relative pl-4">
              {index !== history.length - 1 && (
                <div className="absolute -bottom-6 left-[0.91rem] top-2 w-[2px] bg-muted" />
              )}

              <div className="flex items-center gap-3">
                <div className="relative z-10">
                  <div className="h-3 w-3 translate-x-[-0.375rem] rounded-full border-2 border-primary bg-background" />
                </div>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="cursor-default text-sm text-muted-foreground">
                        {formatDistanceToNow(new Date(state.recorded_at), { addSuffix: true })}
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="right">{state.recorded_at}</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </div>

              <div className="ml-6 mt-2">
                <span className="inline-flex items-center rounded-md bg-muted px-2 py-1">
                  <span className="font-mono text-xs">{state.digest?.slice(0, 12)}</span>
                </span>
              </div>
            </div>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );
}
