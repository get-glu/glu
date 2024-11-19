import { useEffect, useState } from 'react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { getSystem, listPipelines } from '@/services/api';
import { Badge } from '@/components/ui/badge';
import { Pipeline } from '@/components/pipeline';
import { System as SystemType } from '@/types/system';
import { Pipeline as PipelineType } from '@/types/pipeline';
import { ReactFlowProvider } from '@xyflow/react';

export default function System() {
  const [system, setSystem] = useState<SystemType>();
  const [pipelines, setPipelines] = useState<PipelineType[]>();

  useEffect(() => {
    const fetchData = async () => {
      setSystem(await getSystem());
      setPipelines(await listPipelines());
    };
    fetchData();
  }, []);

  return (
    <div className="flex flex-col bg-background">
      <div className="h-18 border-b p-4">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-4">
            <WorkflowIcon className="h-8 w-8" />
            {system && (
              <div>
                <h1 className="text-xl font-bold">{system.name}</h1>
                {system.labels && (
                  <>
                    {Object.keys(system.labels).map((key: string) => {
                      return (
                        <Badge
                          key={`system-label-${key}`}
                          variant={'secondary'}
                          className="whitespace-nowrap text-xs font-light"
                        >
                          {key}: {(system.labels ?? {})[key]}
                        </Badge>
                      );
                    })}
                  </>
                )}
              </div>
            )}
          </div>
          <ThemeToggle />
        </div>
      </div>

      <div className="flex w-full justify-center">
        <div className="flex w-4/5 flex-col">
          {pipelines &&
            pipelines.map((pipeline: PipelineType) => (
              <ReactFlowProvider key={`provider-${pipeline.name}`}>
                <Pipeline key={pipeline.name} pipeline={pipeline} />
              </ReactFlowProvider>
            ))}
        </div>
      </div>
    </div>
  );
}
