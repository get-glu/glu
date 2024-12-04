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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { useState } from 'react';
import { History, Loader2, MoreVertical } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useGetPhaseHistoryQuery, useRollbackPhaseMutation } from '@/services/api';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from './ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from './ui/dialog';
import { State } from '@/types/pipeline';
import { toast } from 'sonner';

interface PhaseHistoryProps {
  pipeline: string;
  phase: string;
}

export function PhaseHistory({ pipeline, phase }: PhaseHistoryProps) {
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);

  const { data: history, isLoading } = useGetPhaseHistoryQuery(
    {
      pipeline,
      phase
    },
    {
      skip: !isHistorySheetOpen,
      refetchOnMountOrArgChange: true
    }
  );

  return (
    <>
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
            {isLoading ? (
              <>
                {[1, 2, 3].map((i) => (
                  <LoadingHistoryItem key={i} />
                ))}
              </>
            ) : (
              history?.map((state, index) => (
                <PhaseHistoryItem
                  key={index}
                  pipeline={pipeline}
                  phase={phase}
                  state={state}
                  index={index}
                />
              ))
            )}
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function LoadingHistoryItem() {
  return (
    <div className="relative pl-4">
      <div className="flex items-center gap-3">
        <Skeleton className="h-3 w-3 rounded-full" />
        <Skeleton className="h-4 w-full" />
      </div>
      <div className="ml-6 mt-2">
        <Skeleton className="h-6 w-48" />
      </div>
    </div>
  );
}

interface PhaseHistoryItemProps {
  pipeline: string;
  phase: string;
  state: State;
  index: number;
}

function PhaseHistoryItem({ pipeline, phase, state, index }: PhaseHistoryItemProps) {
  const [revertDialogOpen, setRevertDialogOpen] = useState(false);
  const [isPerformingRevert, setIsPerformingRevert] = useState(false);
  const [rollbackPhase] = useRollbackPhaseMutation();

  const performRevert = async () => {
    try {
      setIsPerformingRevert(true);
      await rollbackPhase({ pipeline, phase, version: state.version }).unwrap();

      toast.success('Phase reverted');
    } catch (e) {
      console.error(e);

      toast.error('Something went wrong');
    } finally {
      setIsPerformingRevert(false);
      setRevertDialogOpen(false);
    }
  };

  return (
    <>
      <div className="relative pl-4">
        {/* vertical line */}
        {index !== history.length - 1 && (
          <div className="absolute -bottom-6 left-[0.91rem] top-2 w-[2px] bg-muted" />
        )}

        {/* circle */}
        <div className="flex items-center gap-3">
          <div className="relative z-10">
            <div
              className={cn(
                'h-3 w-3 translate-x-[-0.375rem] rounded-full border-2 bg-background',
                index === 0 ? 'border-primary' : 'border-gray-400'
              )}
            />
          </div>

          {/* updated at */}
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

        {/* history card */}
        <div className="ml-6 mt-2 flex justify-between">
          {/* digest and hover */}
          <HoverCard>
            <HoverCardTrigger asChild className="cursor-default">
              <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1">
                <span className="font-mono text-xs">{state.digest?.slice(0, 30)}</span>
              </span>
            </HoverCardTrigger>
            <HoverCardContent className="flex w-full flex-col gap-1">
              <div className="text-sm">
                <span className="text-muted-foreground">Digest: </span>
                {state.digest}
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Recorded At: </span>
                {state.recorded_at}
              </div>
            </HoverCardContent>
          </HoverCard>

          {/* dropdown menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="ml-2 text-muted-foreground">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onSelect={() => console.log('Details action')}>
                Details
              </DropdownMenuItem>
              {index !== 0 && (
                <DropdownMenuItem onSelect={() => setRevertDialogOpen(true)}>
                  Revert
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <Dialog open={revertDialogOpen} onOpenChange={setRevertDialogOpen}>
        <DialogContent className="px-4">
          <DialogHeader>
            <DialogTitle>Revert Phase</DialogTitle>
            <DialogDescription className="flex flex-col gap-4">
              Are you sure you want to revert this phase to this digest?
              <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1">
                <span className="font-mono text-xs">{state.digest}</span>
              </span>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevertDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={performRevert}>
              {isPerformingRevert ? (
                <>
                  <Loader2 className="animate-spin" /> Please Wait{' '}
                </>
              ) : (
                'Revert'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
