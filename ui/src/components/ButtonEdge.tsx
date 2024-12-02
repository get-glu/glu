import { BaseEdge, EdgeLabelRenderer, EdgeProps, getBezierPath } from '@xyflow/react';
import { TooltipProvider, TooltipTrigger, TooltipContent, Tooltip } from '@/components/ui/tooltip';
import { CircleArrowUp, CheckCircle, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { useEdgePerformMutation } from '@/services/api';
import { ANNOTATION_GIT_PROPOSAL_URL } from '@/types/metadata';
import { toast } from 'sonner';
import { PipelineEdge } from '@/types/flow';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

export function ButtonEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  data,
  markerEnd
}: EdgeProps<PipelineEdge>) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition
  });

  const [dialogOpen, setDialogOpen] = useState(false);
  const [isPerforming, setIsPerforming] = useState(false);
  const [performEdge] = useEdgePerformMutation();

  const perform = async () => {
    setIsPerforming(true);

    try {
      let message = <p>Phase promotion scheduled</p>;
      const result = await performEdge({
        pipeline: data?.from.pipeline ?? '',
        from: data?.from.metadata.name ?? '',
        to: data?.to.metadata.name ?? ''
      }).unwrap();
      if (result?.annotations) {
        const proposalURL = result?.annotations[ANNOTATION_GIT_PROPOSAL_URL];
        if (proposalURL) {
          message = (
            <p>
              Phase promotion proposed:{' '}
              <a className="underline" href={proposalURL} target="_blank" rel="noopener noreferrer">
                {proposalURL}
              </a>
            </p>
          );
        }
      }

      toast.success(message);
    } catch (e) {
      console.error(e);

      toast.error('something went wrong');
    } finally {
      setIsPerforming(false);
      setDialogOpen(false);
    }
  };

  return (
    <>
      <BaseEdge path={edgePath} markerEnd={markerEnd} style={style} />
      <EdgeLabelRenderer>
        <div
          className="nodrag nopan pointer-events-auto absolute"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`
          }}
        >
          {data?.can_perform ? (
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
                  <CircleArrowUp
                    className="h-4 w-4 flex-shrink-0 cursor-pointer transition-transform hover:rotate-90 hover:text-green-400"
                    onClick={() => setDialogOpen(true)}
                  />
                </TooltipTrigger>
                <TooltipContent sideOffset={5} className="text-xs">
                  Promote
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
        </div>
      </EdgeLabelRenderer>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Promote Phase</DialogTitle>
            <DialogDescription>Are you sure you want to promote this phase?</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={perform}>
              {isPerforming ? (
                <>
                  <Loader2 className="animate-spin" /> Please Wait{' '}
                </>
              ) : (
                'Promote'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
