import { useEffect } from 'react';
import { ReactFlow, Controls, Background, MarkerType } from '@xyflow/react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { useTheme } from '@/components/theme-provider';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { setFlow, updateNodes, updateEdges } from '@/store/flowSlice';
import { PhaseNode } from '@/components/node';
import { Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineNode, PipelineEdge } from '@/types/flow';
import { getMockPipelines } from '@/services/mockData';
import { GroupNode } from '@/components/group-node';

const nodeTypes = {
  phase: PhaseNode,
  group: GroupNode
};

export default function Workflow() {
  const dispatch = useAppDispatch();
  const { theme } = useTheme();
  const { nodes, edges } = useAppSelector((state) => state.flow);

  useEffect(() => {
    const fetchData = async () => {
      const pipelines = await getMockPipelines();
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
            <div>
              <h1 className="text-xl font-bold">Frontdoor</h1>
              <p className="text-sm text-muted-foreground">https://github.com/flipt-io/frontdoor</p>
            </div>
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
  const PHASE_SPACING_X = 350;
  const PHASE_SPACING_Y = 200;
  const NODE_WIDTH = 200; // Base width of a node

  let currentY = 0;

  pipelines.forEach((pipeline, pipelineIndex) => {
    // First pass: calculate max depth and count phases per column
    const phasesByColumn: { [key: number]: number } = {};
    let maxDepth = 0;

    pipeline.phases.forEach((phase) => {
      const depth = getPhaseDepth(phase, pipeline.phases);
      maxDepth = Math.max(maxDepth, depth);
      const xPosition = depth * PHASE_SPACING_X;
      phasesByColumn[xPosition] = (phasesByColumn[xPosition] || 0) + 1;
    });

    const maxPhasesInColumn = Math.max(...Object.values(phasesByColumn), 1);
    const pipelineHeight = maxPhasesInColumn * PHASE_SPACING_Y + PIPELINE_VERTICAL_PADDING;

    // Calculate required width based on deepest node plus padding
    const requiredWidth = maxDepth * PHASE_SPACING_X + NODE_WIDTH + 125; // 125px for padding

    // Add group node
    const groupNode: PipelineNode = {
      id: pipeline.id,
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
      const xPosition = getPhaseDepth(phase, pipeline.phases) * PHASE_SPACING_X + 50;
      columnCount[xPosition] = (columnCount[xPosition] || 0) + 1;
      const yPosition =
        PIPELINE_VERTICAL_PADDING / 2 + (columnCount[xPosition] - 1) * PHASE_SPACING_Y;

      const node: PipelineNode = {
        id: phase.id,
        type: 'phase',
        position: { x: xPosition, y: yPosition },
        parentId: pipeline.id,
        data: {
          label: phase.name,
          type: phase.type,
          labels: phase.labels
        }
      };
      nodes.push(node);

      if (phase.dependsOn) {
        edges.push({
          id: `edge-${phase.dependsOn}-${phase.id}`,
          target: phase.dependsOn,
          source: phase.id
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
function getPhaseDepth(phase: Pipeline['phases'][0], allPhases: Pipeline['phases']): number {
  if (!phase.dependsOn) return 0;

  const parentPhase = allPhases.find((p) => p.id === phase.dependsOn);
  if (!parentPhase) return 0;

  return 1 + getPhaseDepth(parentPhase, allPhases);
}
