import { Handle, NodeProps, Position } from '@xyflow/react';
import { Package, GitBranch, CircleArrowUp } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { PhaseNode as PhaseNodeType } from '@/types/flow';
import * as Tooltip from '@radix-ui/react-tooltip';
import { promotePhase } from '@/services/api';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { useState } from 'react';

const PhaseNode = ({ data }: NodeProps<PhaseNodeType>) => {
  const getIcon = () => {
    switch (data.source_type ?? '') {
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
  };

  return (
    <div className="relative min-h-[80px] min-w-[120px] rounded-lg border bg-background p-4 shadow-lg">
      <Handle type="source" position={Position.Right} style={{ right: -8 }} />

      <div className="flex items-center gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          {getIcon()}
          <span className="truncate text-sm font-medium">{data.name}</span>
        </div>
        {data.depends_on && data.depends_on !== '' && (
          <Tooltip.Provider>
            <Tooltip.Root>
              <Tooltip.Trigger asChild>
                <CircleArrowUp
                  className="ml-2 h-4 w-4 flex-shrink-0 cursor-pointer transition-transform hover:rotate-90 hover:text-green-600"
                  onClick={() => setDialogOpen(true)}
                />
              </Tooltip.Trigger>
              <Tooltip.Portal>
                <Tooltip.Content
                  className="data-[state=delayed-open]:data-[side=bottom]:animate-slideUpAndFade data-[state=delayed-open]:data-[side=left]:animate-slideRightAndFade data-[state=delayed-open]:data-[side=right]:animate-slideLeftAndFade data-[state=delayed-open]:data-[side=top]:animate-slideDownAndFadet select-none rounded bg-white px-[15px] py-2.5 text-sm leading-none shadow-[hsl(206_22%_7%_/_35%)_0px_10px_38px_-10px,_hsl(206_22%_7%_/_20%)_0px_10px_20px_-15px] will-change-[transform,opacity]"
                  sideOffset={5}
                >
                  Promote
                  <Tooltip.Arrow className="fill-white" />
                </Tooltip.Content>
              </Tooltip.Portal>
            </Tooltip.Root>
          </Tooltip.Provider>
        )}
      </div>

      <div className="mt-2 font-mono text-xs text-muted-foreground">
        {data.value?.digest?.slice(-12)}
      </div>

      <div className="mt-2 flex w-full flex-col">
        {data.labels &&
          Object.entries(data.labels).length > 0 &&
          Object.entries(data.labels).map(([key, value]) => (
            <div key={`${key}-${value}`} className="mb-2 flex">
              <Badge
                key={`${key}-${value}`}
                className={`whitespace-nowrap text-xs font-light ${getLabelColor(key, value)}`}
              >
                {key}: {value}
              </Badge>
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

function getLabelColor(key: string, value: string): string {
  const hash = `${key}:${value}`.split('').reduce((acc, char) => {
    return char.charCodeAt(0) + ((acc << 5) - acc);
  }, 0);

  const colors = [
    'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900 dark:hover:bg-red-800 dark:text-red-200',
    'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900 dark:hover:bg-blue-800 dark:text-blue-200',
    'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900 dark:hover:bg-green-800 dark:text-green-200',
    'bg-yellow-100 text-yellow-800 hover:bg-yellow-200 dark:bg-yellow-900 dark:hover:bg-yellow-800 dark:text-yellow-200',
    'bg-purple-100 text-purple-800 hover:bg-purple-200 dark:bg-purple-900 dark:hover:bg-purple-800 dark:text-purple-200',
    'bg-pink-100 text-pink-800 hover:bg-pink-200 dark:bg-pink-900 dark:hover:bg-pink-800 dark:text-pink-200',
    'bg-indigo-100 text-indigo-800 hover:bg-indigo-200 dark:bg-indigo-900 dark:hover:bg-indigo-800 dark:text-indigo-200',
    'bg-orange-100 text-orange-800 hover:bg-orange-200 dark:bg-orange-900 dark:hover:bg-orange-800 dark:text-orange-200',
    'bg-gray-100 text-gray-800 hover:bg-gray-200 dark:bg-gray-900 dark:hover:bg-gray-800 dark:text-gray-200'
  ];

  return colors[Math.abs(hash) % colors.length];
}

export { PhaseNode };
