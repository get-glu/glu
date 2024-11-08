import { useEffect } from 'react';
import { ReactFlow, Controls, Background, MarkerType } from '@xyflow/react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { useTheme } from '@/components/theme-provider';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { setFlow, updateNodes, updateEdges } from '@/store/flowSlice';
import { ControllerNode } from '@/components/node';
import { Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineNode, PipelineEdge } from '@/types/flow';
import { GroupNode } from '@/components/group-node';
import { listPipelines } from '@/services/api';

const nodeTypes = {
  controller: ControllerNode,
  group: GroupNode
};

export default function Workflow() {
  const dispatch = useAppDispatch();
  const { theme } = useTheme();
  const { nodes, edges } = useAppSelector((state) => state.flow);

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
  const CONTROLLER_SPACING_X = 350;
  const CONTROLLER_SPACING_Y = 200;
  const NODE_WIDTH = 200; // Base width of a node

  let currentY = 0;

  pipelines.forEach((pipeline, pipelineIndex) => {
    // First pass: calculate max depth and count controllers per column
    const controllersByColumn: { [key: number]: number } = {};
    let maxDepth = 0;

    pipeline.controllers.forEach((controller) => {
      const depth = getNodeDepth(controller, pipeline.controllers);
      maxDepth = Math.max(maxDepth, depth);
      const xPosition = depth * CONTROLLER_SPACING_X;
      controllersByColumn[xPosition] = (controllersByColumn[xPosition] || 0) + 1;
    });

    const maxPhasesInColumn = Math.max(...Object.values(controllersByColumn), 1);
    const pipelineHeight = maxPhasesInColumn * CONTROLLER_SPACING_Y + PIPELINE_VERTICAL_PADDING;

    // Calculate required width based on deepest node plus padding
    const requiredWidth = maxDepth * CONTROLLER_SPACING_X + NODE_WIDTH + 125; // 125px for padding

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

    pipeline.controllers.forEach((controller) => {
      const xPosition = getNodeDepth(controller, pipeline.controllers) * CONTROLLER_SPACING_X + 50;
      columnCount[xPosition] = (columnCount[xPosition] || 0) + 1;
      const yPosition =
        PIPELINE_VERTICAL_PADDING / 2 + (columnCount[xPosition] - 1) * CONTROLLER_SPACING_Y;

      const node: PipelineNode = {
        id: controller.name,
        type: 'controller',
        position: { x: xPosition, y: yPosition },
        parentId: pipeline.name,
        data: {
          name: controller.name,
          labels: controller.labels || {}
        }
      };
      nodes.push(node);

      if (controller.depends_on) {
        edges.push({
          id: `edge-${controller.depends_on}-${controller.name}`,
          target: controller.depends_on,
          source: controller.name
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

// Helper function to calculate the depth of a controller based on its dependencies
function getNodeDepth(
  controller: Pipeline['controllers'][0],
  allControllers: Pipeline['controllers']
): number {
  if (!controller.depends_on) return 0;

  const parentController = allControllers.find((c) => c.name === controller.depends_on);
  if (!parentController) return 0;

  return 1 + getNodeDepth(parentController, allControllers);
}
