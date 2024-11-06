import { useEffect } from 'react';
import { ReactFlow, Controls, Background } from '@xyflow/react';
import { Package } from 'lucide-react';
import '@xyflow/react/dist/style.css';
import { ThemeToggle } from '@/components/theme-toggle';
import { useTheme } from '@/components/theme-provider';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { fetchPipeline } from '@/store/pipelineSlice';
import { setFlow, updateNodes, updateEdges, addEdge } from '@/store/flowSlice';
import { PhaseNode, ResourceNode } from '@/components/node';
import { Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineNode, PipelineEdge } from '@/types/flow';

const nodeTypes = {
  phase: PhaseNode,
  resource: ResourceNode
};

export default function Workflow() {
  const dispatch = useAppDispatch();
  const { theme } = useTheme();
  const { data: pipeline, loading, error } = useAppSelector((state) => state.pipeline);
  const { nodes, edges } = useAppSelector((state) => state.flow);

  useEffect(() => {
    dispatch(fetchPipeline('foundation'));
  }, [dispatch]);

  useEffect(() => {
    if (pipeline) {
      const flow = transformPipeline(pipeline);
      dispatch(setFlow({ nodes: flow.nodes, edges: flow.edges }));
    }
  }, [pipeline, dispatch]);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  if (!pipeline) return null;

  return (
    <div className="flex h-screen flex-col bg-background">
      <div className="h-18 border-b p-4">
        <div className="mx-auto flex max-w-7xl items-center justify-between">
          <div className="flex items-center gap-4">
            <Package className="h-8 w-8" />
            <div>
              <h1 className="text-xl font-bold">{pipeline.name}</h1>
              <p className="text-sm text-muted-foreground">{pipeline.name}</p>
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
          onConnect={(connection) => dispatch(addEdge(connection))}
          fitView
          colorMode={theme}
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}

export function transformPipeline(serverPipeline: Pipeline): FlowPipeline {
  const nodes: PipelineNode[] = [];
  const edges: PipelineEdge[] = [];

  let xOffset = 0;
  const PHASE_SPACING = 500; // Horizontal spacing between phase groups
  const RESOURCE_SPACING = 100; // Vertical spacing between resources
  const PHASE_TO_RESOURCE_SPACING = 200; // Horizontal spacing between phase and its resources

  serverPipeline.phases.forEach((phase, phaseIndex) => {
    // Add phase node
    const phaseNode: PipelineNode = {
      id: `phase-${phase.name}`,
      type: 'phase',
      position: { x: xOffset, y: 0 },
      data: {
        label: phase.name,
        resources: phase.resources,
        environment: phase.name,
        color: 'bg-primary/10'
      }
    };
    nodes.push(phaseNode);

    // Add resource nodes and edges to the right of each phase
    phase.resources.forEach((resourceName, resourceIndex) => {
      const resourceNode: PipelineNode = {
        id: `resource-${resourceName}-${phase.name}`,
        type: 'resource',
        position: {
          x: xOffset + PHASE_TO_RESOURCE_SPACING, // Position resources to the right of phase
          y: resourceIndex * RESOURCE_SPACING // Stack resources vertically
        },
        data: {
          label: resourceName,
          type: 'git'
        }
      };
      nodes.push(resourceNode);

      edges.push({
        id: `edge-${phaseNode.id}-${resourceNode.id}`,
        source: phaseNode.id,
        target: resourceNode.id
      });
    });

    // Move to next phase group, accounting for phase width and resources
    xOffset += PHASE_SPACING;
  });

  return {
    nodes,
    edges
  };
}
