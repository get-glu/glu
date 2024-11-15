import { GroupNode } from '@/types/flow';
import { NodeProps } from '@xyflow/react';

export function GroupNode({ data }: NodeProps<GroupNode>) {
  return (
    <div className="min-h-[200px] min-w-[800px] rounded-lg p-4">
      <div className="absolute -top-3 left-2 bg-background px-2 text-lg font-medium text-muted-foreground">
        {data.labels?.name}
      </div>
    </div>
  );
}
