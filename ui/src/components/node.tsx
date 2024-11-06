import { Handle, Position } from '@xyflow/react';
import { GitBranch, ImageIcon } from 'lucide-react';
import { Input } from '@/components/ui/input';

const PhaseNode = ({ data }: { data: any }) => (
  <div className="min-w-[150px] rounded-lg border bg-background px-4 py-2 shadow-lg">
    <Handle type="target" position={Position.Left} />
    <div className="flex items-center justify-between gap-2">
      <span className="text-sm font-medium">{data.label}</span>
      <span className={`rounded px-2 py-0.5 text-xs ${data.color || 'bg-primary/10'}`}>
        {data.environment || data.label}
      </span>
    </div>
    {data.resources && (
      <div className="mt-2 space-y-1">
        {data.resources.map((resource: string) => (
          <div key={resource} className="rounded bg-muted p-2 text-xs">
            <div className="font-mono">{resource}</div>
          </div>
        ))}
      </div>
    )}
    <Handle type="source" position={Position.Right} />
  </div>
);

const ResourceNode = ({ data }: { data: any }) => (
  <div className="min-w-[200px] rounded-lg border bg-background p-4 shadow-lg">
    <Handle type="source" position={Position.Right} />
    <div className="flex items-center gap-2">
      {data.type === 'git' ? <GitBranch className="h-4 w-4" /> : <ImageIcon className="h-4 w-4" />}
      <span className="text-sm font-medium">{data.label}</span>
    </div>
    {data.url && <Input className="mt-2 text-xs" placeholder="URL" value={data.url} readOnly />}
  </div>
);

export { PhaseNode, ResourceNode };
