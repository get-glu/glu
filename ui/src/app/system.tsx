import { useEffect, useState } from 'react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { getSystem, listPipelines } from '@/services/api';
import { Badge } from '@/components/ui/badge';
import { Pipeline as PipelineComponent } from '@/app/pipeline';
import { System } from '@/types/system';
import { Pipeline } from '@/types/pipeline';
import { ReactFlowProvider } from '@xyflow/react';

export default function System() {
  const [system, setSystem] = useState<System>();
  const [pipelines, setPipelines] = useState<Pipeline[]>();

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
                        <Badge key={`system-label-${key}`} variant={'secondary'}>
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
        <div className="flex w-3/5 flex-col">
          {pipelines &&
            pipelines.map((pipeline: Pipeline) => (
              <ReactFlowProvider key={`provider-${pipeline.name}`}>
                <PipelineComponent key={pipeline.name} pipeline={pipeline}></PipelineComponent>
              </ReactFlowProvider>
            ))}
        </div>
      </div>
    </div>
  );
}
