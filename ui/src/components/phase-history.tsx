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
import { Github, History, MoreVertical } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useGetPhaseHistoryQuery } from '@/services/api';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from './ui/button';
import { State } from '@/types/pipeline';
import { PhaseStateDetails } from './phase-state-details';
import { PhaseRollbackDialog } from './phase-rollback-dialog';
import { ANNOTATION_GIT_COMMIT_URL } from '@/types/metadata';

interface PhaseHistoryProps {
  pipelineId: string;
  phaseId: string;
}

export function PhaseHistory({ pipelineId, phaseId }: PhaseHistoryProps) {
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);

  const { data: history, isLoading } = useGetPhaseHistoryQuery(
    {
      pipeline: pipelineId,
      phase: phaseId
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
          <SheetHeader className="flex flex-col gap-1 border-b border-b-muted pb-4">
            <SheetTitle>{phaseId}</SheetTitle>
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
                  pipelineId={pipelineId}
                  phaseId={phaseId}
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
  pipelineId: string;
  phaseId: string;
  state: State;
  index: number;
}

function PhaseHistoryItem({ pipelineId, phaseId, state, index }: PhaseHistoryItemProps) {
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);

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
              <TooltipContent side="right">
                {new Date(state.recorded_at).toUTCString()}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>

        {/* history card */}
        <div className="ml-6 mt-2 flex justify-between">
          {/* digest and hover */}
          <div className="flex flex-col gap-2">
            <HoverCard>
              <HoverCardTrigger asChild className="cursor-default">
                <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1">
                  <span className="font-mono text-xs">{state.digest?.slice(0, 30)}</span>
                </span>
              </HoverCardTrigger>
              <HoverCardContent className="flex w-full flex-col gap-1">
                <div className="text-sm">
                  <span className="text-foreground">Recorded At: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 text-xs">
                    {new Date(state.recorded_at).toUTCString()}
                  </span>
                </div>
                <div className="text-sm">
                  <span className="text-foreground">Digest: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 font-mono text-xs">
                    {state.digest}
                  </span>
                </div>
                <div className="text-sm">
                  <span className="text-foreground">Version: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 font-mono text-xs">
                    {state.version}
                  </span>
                </div>
              </HoverCardContent>
            </HoverCard>

            <div className="group flex flex-row flex-wrap gap-2 hover:underline">
              {state.annotations?.[ANNOTATION_GIT_COMMIT_URL]?.startsWith('https://github.com') && (
                <a
                  href={state.annotations[ANNOTATION_GIT_COMMIT_URL]}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 px-2 py-1 text-xs"
                >
                  <Github className="h-2.5 w-2.5" />
                  Commit
                </a>
              )}
            </div>
          </div>

          {/* dropdown menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="ml-2 text-muted-foreground">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onSelect={() => setDetailsDialogOpen(true)}>
                Details
              </DropdownMenuItem>
              {index !== 0 && (
                <DropdownMenuItem onSelect={() => setRollbackDialogOpen(true)}>
                  Rollback
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <PhaseRollbackDialog
        isOpen={rollbackDialogOpen}
        onClose={() => setRollbackDialogOpen(false)}
        pipelineId={pipelineId}
        phaseId={phaseId}
        state={state}
      />

      <PhaseStateDetails
        isOpen={detailsDialogOpen}
        onClose={() => setDetailsDialogOpen(false)}
        pipelineId={pipelineId}
        phaseId={phaseId}
        state={state}
        latest={index === 0}
      />
    </>
  );
}
