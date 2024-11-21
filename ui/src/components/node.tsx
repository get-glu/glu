import { Handle, NodeProps, Position } from '@xyflow/react';
import { Package, GitBranch, CircleArrowUp, CheckCircle } from 'lucide-react';
import { PhaseNode as PhaseNodeType } from '@/types/flow';
import { promotePhase } from '@/services/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useState } from 'react';
import { ANNOTATION_OCI_IMAGE_URL } from '@/types/metadata';
import { Label } from './label';
import { TooltipProvider, TooltipTrigger, TooltipContent, Tooltip } from '@/components/ui/tooltip';
import { toast } from 'sonner';

const PhaseNode = ({ data }: NodeProps<PhaseNodeType>) => {
  const getIcon = () => {
    switch (data.source.name ?? '') {
      case 'oci':
        return <Package className="h-4 w-4" />;
      default:
        return <GitBranch className="h-4 w-4" />;
    }
  };

  const [dialogOpen, setDialogOpen] = useState(false);

  const promote = async () => {
    setDialogOpen(false);
    await promotePhase(data.pipeline, data.name);
    toast.success('Phase promotion scheduled');
  };

  return (
    <div className="relative min-h-[80px] min-w-[120px] cursor-pointer rounded-lg border bg-background p-4 shadow-lg">
      <Handle type="source" position={Position.Right} style={{ right: -8 }} />

      <div className="flex items-center gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {getIcon()}
          <span className="truncate text-sm font-medium">{data.name}</span>
        </div>
        {data.depends_on && data.depends_on !== '' && (
          <>
            {data.synced ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <CheckCircle className="ml-2 h-4 w-4 flex-shrink-0 text-green-600" />
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
                      className="ml-2 h-4 w-4 flex-shrink-0 cursor-pointer transition-transform hover:rotate-90 hover:text-green-600"
                      onClick={() => setDialogOpen(true)}
                    />
                  </TooltipTrigger>
                  <TooltipContent sideOffset={5} className="text-xs">
                    Promote
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </>
        )}
      </div>

      <div className="mt-2 flex items-center gap-2 text-xs">
        <span>Digest:</span>
        <span className="font-mono text-xs text-muted-foreground">{data.digest?.slice(-12)}</span>
      </div>

      {data.source.annotations?.[ANNOTATION_OCI_IMAGE_URL] && (
        <div className="mt-2 flex items-center gap-2 text-xs">
          <span>Image:</span>
          <a
            href={`https://${data.source.annotations[ANNOTATION_OCI_IMAGE_URL]}`}
            target="_blank"
            rel="noopener noreferrer"
            className="truncate font-mono text-xs text-muted-foreground hover:text-primary hover:underline"
          >
            {data.source.annotations[ANNOTATION_OCI_IMAGE_URL]}
          </a>
        </div>
      )}

      <div className="mt-2 flex w-full flex-col">
        {data.labels &&
          Object.entries(data.labels).length > 0 &&
          Object.entries(data.labels).map(([key, value]) => (
            <div key={`${key}-${value}`} className="mb-2 flex">
              <Label labelKey={key} value={value} />
            </div>
          ))}
      </div>

      <Handle type="target" position={Position.Left} style={{ left: -8 }} />

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
            <Button onClick={promote}>Promote</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export { PhaseNode };
