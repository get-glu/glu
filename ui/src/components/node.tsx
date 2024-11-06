import { Handle, Position } from '@xyflow/react';
import { Package, GitBranch } from 'lucide-react';

const PhaseNode = ({ data }: { data: any }) => {
  const getIcon = () => {
    switch (data.type) {
      case 'oci':
        return <Package className="h-4 w-4" />;
      default:
        return <GitBranch className="h-4 w-4" />;
    }
  };

  return (
    <div className="min-w-[150px] rounded-lg border bg-background px-4 py-2 shadow-lg">
      <Handle type="target" position={Position.Left} />
      <div className="flex items-center gap-2">
        {getIcon()}
        <span className="text-sm font-medium">{data.label}</span>
      </div>
      <Handle type="source" position={Position.Right} />
    </div>
  );
};

export { PhaseNode };
