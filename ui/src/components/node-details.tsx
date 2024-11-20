import { Sheet, SheetContent } from '@/components/ui/sheet';
import { PhaseNode } from '@/types/flow';
import { Badge } from './ui/badge';
import { Package, GitBranch } from 'lucide-react';
import { getLabelColor } from '@/lib/utils';

interface NodeDetailsProps {
  node: PhaseNode | null;
  onClose: () => void;
}

export function NodeDetails({ node, onClose }: NodeDetailsProps) {
  if (!node) return null;

  const getIcon = () => {
    switch (node.data.source.name ?? '') {
      case 'oci':
        return <Package className="h-4 w-4" />;
      default:
        return <GitBranch className="h-4 w-4" />;
    }
  };

  return (
    <Sheet open={!!node} onOpenChange={() => onClose()}>
      <SheetContent side="bottom" className="h-1/3">
        <div className="flex h-full flex-col">
          <div className="flex items-center gap-2 pb-4">
            {getIcon()}
            <h2 className="text-lg font-semibold">{node.data.name}</h2>
          </div>

          <div className="grid flex-1 grid-cols-2 gap-4 overflow-auto">
            <div className="space-y-4">
              <div>
                <h3 className="text-sm font-medium">Details</h3>
                <div className="mt-2 space-y-2">
                  <div className="text-sm">
                    <span className="text-muted-foreground">Pipeline: </span>
                    {node.data.pipeline}
                  </div>
                  <div className="text-sm">
                    <span className="text-muted-foreground">Depends on: </span>
                    {node.data.depends_on || 'None'}
                  </div>
                  <div className="text-sm">
                    <span className="text-muted-foreground">Digest: </span>
                    <span className="font-mono">{node.data.digest}</span>
                  </div>
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <h3 className="text-sm font-medium">Labels</h3>
                <div className="mt-2 flex flex-wrap gap-2">
                  {node.data.labels &&
                    Object.entries(node.data.labels).map(([key, value]) => (
                      <Badge
                        key={`${key}-${value}`}
                        className={`whitespace-nowrap text-xs font-light ${getLabelColor(key, value)}`}
                      >
                        {key}: {value}
                      </Badge>
                    ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
