import { useCallback, useEffect, useState } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  MarkerType,
  useReactFlow,
  useEdgesState,
  useNodesState,
  useNodesInitialized
} from '@xyflow/react';
import { WorkflowIcon } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { useTheme } from '@/components/theme-provider';
import { PhaseNode as PhaseNodeComponent } from '@/components/node';
import { Phase, Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineEdge, PhaseNode, GroupNode, PipelineNode } from '@/types/flow';
import { GroupNode as GroupNodeComponent } from '@/components/group-node';
import { getSystem, listPipelines } from '@/services/api';
import { Badge } from '@/components/ui/badge';
import { System } from '@/types/system';
import Dagre from '@dagrejs/dagre';

const nodeTypes = {
  phase: PhaseNodeComponent,
  group: GroupNodeComponent
};

const initialNodes: PipelineNode[] = [];
const initialEdges: PipelineEdge[] = [];

export default function Workflow() {
  const { theme } = useTheme();
  const [system, setSystem] = useState<System>();

  const { fitView } = useReactFlow<PipelineNode, PipelineEdge>();
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const nodesInitialized = useNodesInitialized();
  const [initLayoutFinished, setInitLayoutFinished] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      setSystem(await getSystem());
    };
    fetchData();
  }, [system]);

  useEffect(() => {
    const fetchData = async () => {
      const pipelines = await listPipelines();
      const flow = getElements(pipelines);

      setNodes([...flow.nodes]);
      setEdges([...flow.edges]);
    };

    fetchData();
  }, []);

  useEffect(() => {
    if (nodesInitialized && !initLayoutFinished) {
      const flow = getLayoutedElements({ nodes, edges });

      setNodes([...flow.nodes]);
      setEdges([...flow.edges]);

      window.requestAnimationFrame(async () => {
        await fitView();

        if (!initLayoutFinished) {
          setInitLayoutFinished(true);
        }
      });
    }
  }, [nodesInitialized, initLayoutFinished, nodes, edges, setNodes, setEdges, fitView]);

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
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
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

function getLayoutedElements(pipeline: FlowPipeline) {
  const g = new Dagre.graphlib.Graph({ compound: true }).setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'LR' });

  pipeline.edges.forEach((edge) => g.setEdge(edge.source, edge.target));
  pipeline.nodes.forEach((node) => {
    console.log('measured:', node.id, node.parentId, node.measured);
    g.setNode(node.id, {
      ...node,
      width: node.measured?.width ?? 0,
      height: node.measured?.height ?? 0
    });
    if (node.parentId) {
      g.setParent(node.id, node.parentId);
    }
  });

  Dagre.layout(g);

  return {
    nodes: pipeline.nodes.map((node) => {
      const position = g.node(node.id);

      // We are shifting the dagre node position (anchor=center center) to the top left
      // so it matches the React Flow node anchor point (top left).
      const x = position.x - (node.measured?.width ?? 0) / 2;
      const y = position.y - (node.measured?.height ?? 0) / 2;

      return { ...node, position: { x, y } };
    }),
    edges: pipeline.edges
  };
}

function getElements(pipelines: Pipeline[]): FlowPipeline {
  const nodes: PipelineNode[] = [];
  const edges: PipelineEdge[] = [];
  const PIPELINE_VERTICAL_PADDING = 100;
  const PIPELINE_SPACING = 100;
  const PHASE_SPACING_Y = 200;
  const PHASE_PADDING_X = 50;
  const PHASE_WIDTH = 250;

  let currentY = 0;

  pipelines.forEach((pipeline, pipelineIndex) => {
    // First pass: calculate max depth and count phases per column
    const phasesByColumn: { [key: number]: number } = {};
    let maxDepth = 0;

    pipeline.phases.forEach((phase) => {
      const depth = getNodeDepth(phase, pipeline.phases);
      maxDepth = Math.max(maxDepth, depth);
      phasesByColumn[depth] = (phasesByColumn[depth] || 0) + 1;
    });

    const maxPhasesInColumn = Math.max(...Object.values(phasesByColumn), 1);
    const pipelineHeight = maxPhasesInColumn * PHASE_SPACING_Y + PIPELINE_VERTICAL_PADDING;

    // Calculate required width based on deepest node plus padding
    const requiredWidth = (maxDepth + 1) * (2 * PHASE_PADDING_X + PHASE_WIDTH);

    // Add group node
    const groupNode: GroupNode = {
      id: pipeline.name,
      type: 'group',
      position: { x: 0, y: currentY },
      data: { labels: { name: pipeline.name } },
      style: {
        // width: requiredWidth,
        // height: pipelineHeight
      }
    };
    nodes.push(groupNode);

    const columnCount: { [key: number]: number } = {};

    pipeline.phases.forEach((phase) => {
      const depth = getNodeDepth(phase, pipeline.phases);
      columnCount[depth] = (columnCount[depth] || 0) + 1;

      const xPosition = depth * (PHASE_WIDTH + PHASE_PADDING_X) + PHASE_PADDING_X;

      const yPosition = PIPELINE_VERTICAL_PADDING / 2 + (columnCount[depth] - 1) * PHASE_SPACING_Y;

      const phaseId = `${pipeline.name}-${phase.name}`;
      const node: PhaseNode = {
        id: phaseId,
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
          id: `edge-${pipeline.name}-${phase.depends_on}-${phase.name}`,
          source: `${pipeline.name}-${phase.depends_on}`,
          target: phaseId
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
function getNodeDepth(phase: Phase, allPhases: Phase[]): number {
  if (!phase.depends_on) return 0;

  const parentPhase = allPhases.find((c) => c.name === phase.depends_on);
  if (!parentPhase) return 0;

  return 1 + getNodeDepth(parentPhase, allPhases);
}
