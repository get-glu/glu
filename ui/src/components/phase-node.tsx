import { Handle, NodeProps, Position } from '@xyflow/react';

import { PhaseNode as PhaseNodeType } from '@/types/flow';
import { ANNOTATION_OCI_IMAGE_URL } from '@/types/metadata';
import { Label } from './label';
import { getSourceIcon } from '@/types/icons';

const PhaseNode = ({ data: phase }: NodeProps<PhaseNodeType>) => {
  const Icon = getSourceIcon(phase.descriptor.source);

  return (
    <div className="relative min-h-[80px] min-w-[120px] cursor-pointer rounded-lg border bg-background p-4 shadow-lg">
      <Handle type="source" position={Position.Right} style={{ right: -8 }} />

      <div className="flex items-center gap-2">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <Icon className="h-4 w-4" />
          <span className="truncate text-sm font-medium">{phase.descriptor.metadata.name}</span>
        </div>
      </div>

      {phase.descriptor.metadata.annotations?.[ANNOTATION_OCI_IMAGE_URL] && (
        <div className="mt-2 flex items-center gap-2 text-xs">
          <span>Image:</span>
          <a
            href={`https://${phase.descriptor.metadata.annotations[ANNOTATION_OCI_IMAGE_URL]}`}
            target="_blank"
            rel="noopener noreferrer"
            className="truncate font-mono text-xs text-muted-foreground hover:text-primary hover:underline"
          >
            {phase.descriptor.metadata.annotations[ANNOTATION_OCI_IMAGE_URL]}
          </a>
        </div>
      )}

      <div className="mt-4 flex w-full flex-wrap gap-2">
        {phase.descriptor.metadata.labels &&
          Object.keys(phase.descriptor.metadata.labels).length > 0 &&
          Object.entries(phase.descriptor.metadata.labels)
            .slice(0, 3)
            .map(([key, value]) => (
              <div key={`${key}-${value}`} className="mb-2 flex">
                <Label labelKey={key} value={value} />
              </div>
            ))}
      </div>

      {phase.status && Object.keys(phase.status).length > 0 && (
        <div className="mt-4 flex w-full flex-wrap border border-muted p-2">
          <div className="relative -top-[22px] left-0 w-0">
            <span className="text-xs">status:</span>
          </div>
          {Object.entries(phase.status).map(([key, value]) => (
            <div key={`status-${key}-${value}`} className="mt-2 flex text-xs">
              <span className="mr-2 text-muted-foreground">{key}: </span>
              <span className="max-w-[150px] overflow-x-scroll whitespace-nowrap">{value}</span>
            </div>
          ))}
        </div>
      )}

      <Handle type="target" position={Position.Left} style={{ left: -8 }} />
    </div>
  );
};

export { PhaseNode };
