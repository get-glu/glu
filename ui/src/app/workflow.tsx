import { useEffect, useState } from 'react';
import { ReactFlow, Controls, Background, MarkerType } from '@xyflow/react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { useTheme } from '@/components/theme-provider';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { setFlow, updateNodes, updateEdges } from '@/store/flowSlice';
import { PhaseNode } from '@/components/node';
import { Phase, Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineNode, PipelineEdge } from '@/types/flow';
import { GroupNode } from '@/components/group-node';
import { getSystem, listPipelines } from '@/services/api';
import { Badge } from '@/components/ui/badge';
import { System } from '@/types/system';
import { getNodesBounds } from 'reactflow';

const nodeTypes = {
  phase: PhaseNode,
  group: GroupNode
};

export default function Workflow() {
  const dispatch = useAppDispatch();
  const { theme } = useTheme();
  const { nodes, edges } = useAppSelector((state) => state.flow);
  const [system, setSystem] = useState<System | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      setSystem(await getSystem());
    };
    fetchData();
  }, []);

  useEffect(() => {
    const fetchData = async () => {
      const pipelines = await listPipelines();
      const flow = transformPipelines(pipelines);
      dispatch(setFlow({ nodes: flow.nodes, edges: flow.edges }));
    };
    fetchData();
  }, [dispatch]);

  return (
    <div className="flex h-screen flex-col bg-background">
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

      <div className="flex-1">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onNodesChange={(changes) => dispatch(updateNodes(changes))}
          onEdgesChange={(changes) => dispatch(updateEdges(changes))}
          fitView
          colorMode={theme}
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{
            markerEnd: {
              type: MarkerType.Arrow,
              width: 20,
              height: 20,
              color: 'currentColor'
            },
            animated: true,
            selectable: false,
            style: {
              strokeWidth: 2
            }
          }}
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}

export function transformPipelines(pipelines: Pipeline[]): FlowPipeline {
  const nodes: PipelineNode[] = [];
  const edges: PipelineEdge[] = [];
  const PIPELINE_VERTICAL_PADDING = 100;
  const PIPELINE_SPACING = 100;
  const PHASE_SPACING_Y = 200;
  const PHASE_PADDING_X = 50;

  let currentY = 0;

  pipelines.forEach((pipeline, pipelineIndex) => {
    // First pass: calculate max depth and count phases per column
    const phasesByColumn: { [key: number]: number } = {};
    const maxWidthByColumn: { [key: number]: number } = {};
    let maxDepth = 0;

    pipeline.phases.forEach((phase) => {
      const depth = getNodeDepth(phase, pipeline.phases);
      maxDepth = Math.max(maxDepth, depth);
      phasesByColumn[depth] = (phasesByColumn[depth] || 0) + 1;
      maxWidthByColumn[depth] = Math.max(maxWidthByColumn[depth] || 0, getNodeWidth(phase));
    });

    const maxPhasesInColumn = Math.max(...Object.values(phasesByColumn), 1);
    const pipelineHeight = maxPhasesInColumn * PHASE_SPACING_Y + PIPELINE_VERTICAL_PADDING;

    // Calculate required width based on deepest node plus padding
    // const requiredWidth = ((maxDepth + 1) * (3 * PHASE_PADDING_X + PHASE_WIDTH));
    const requiredWidth = Object.values(maxWidthByColumn).reduce((prev, curr) => {
      return prev + curr + 2 * PHASE_PADDING_X;
    }, 0);

    // Add group node
    const groupNode: PipelineNode = {
      id: pipeline.name,
      type: 'group',
      position: { x: 0, y: currentY },
      data: { labels: { name: pipeline.name } },
      style: {
        width: requiredWidth,
        height: pipelineHeight
      }
    };
    nodes.push(groupNode);

    const columnCount: { [key: number]: number } = {};

    pipeline.phases.forEach((phase) => {
      const depth = getNodeDepth(phase, pipeline.phases);
      columnCount[depth] = (columnCount[depth] || 0) + 1;

      // const xPosition = depth * (PHASE_WIDTH + (2 * PHASE_PADDING_X)) + ((depth + 1) * PHASE_PADDING_X);
      // WTF Typescript - Object.keys (even when object is typed as {[num]:num}) thinks it returns string[]
      const xPosition =
        (Object.keys(maxWidthByColumn) as unknown as number[])
          .sort()
          .slice(0, depth)
          .reduce((prev, curr) => {
            return prev + (maxWidthByColumn[curr] || 0) + 2 * PHASE_PADDING_X;
          }, 0) + PHASE_PADDING_X;

      const yPosition = PIPELINE_VERTICAL_PADDING / 2 + (columnCount[depth] - 1) * PHASE_SPACING_Y;

      const node: PipelineNode = {
        id: phase.name,
        type: 'phase',
        position: { x: xPosition, y: yPosition },
        parentId: pipeline.name,
        data: {
          name: phase.name,
          labels: phase.labels || {}
        },
        extent: 'parent'
      };
      nodes.push(node);

      if (phase.depends_on) {
        edges.push({
          id: `edge-${phase.depends_on}-${phase.name}`,
          source: phase.depends_on,
          target: phase.name
        });
      }
    });

    // Only add spacing if this isn't the last pipeline
    if (pipelineIndex < pipelines.length - 1) {
      currentY += pipelineHeight + PIPELINE_SPACING;
    } else {
      currentY += pipelineHeight;
    }
  });

  return { nodes, edges };
}

// Helper function to calculate the depth of a phase based on its dependencies
function getNodeDepth(phase: Pipeline['phases'][0], allPhases: Pipeline['phases']): number {
  if (!phase.depends_on) return 0;

  const parentPhase = allPhases.find((c) => c.name === phase.depends_on);
  if (!parentPhase) return 0;

  return 1 + getNodeDepth(parentPhase, allPhases);
}

function getNodeWidth(phase: Phase): number {
  let entries = [20, phase.name.length];
  entries.push(
    ...Object.entries(phase.labels || {}).map(([key, value]): number => {
      return `${key}: ${value}`.length;
    })
  );
  return Math.max(...entries) * 10;
}
