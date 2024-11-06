import { useEffect } from 'react';
import { ReactFlow, Controls, Background } from '@xyflow/react';
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
            type: 'smoothstep'
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
  const PIPELINE_SPACING = 300;
  const PHASE_SPACING = 250;

  pipelines.forEach((pipeline, pipelineIndex) => {
    // Add group node
    const groupNode: PipelineNode = {
      id: pipeline.id,
      type: 'group',
      position: { x: 0, y: pipelineIndex * PIPELINE_SPACING },
      data: { label: pipeline.name },
      style: {
        width: pipeline.phases.length * PHASE_SPACING + 100,
        height: 200
      }
    };
    nodes.push(groupNode);

    // Add phase nodes
    pipeline.phases.forEach((phase, phaseIndex) => {
      const node: PipelineNode = {
        id: phase.id,
        type: 'phase',
        position: { x: phaseIndex * PHASE_SPACING + 50, y: 80 },
        parentId: pipeline.id,
        data: {
          label: phase.name,
          type: phase.type
        }
      };
      nodes.push(node);

      if (phase.dependsOn) {
        edges.push({
          id: `edge-${phase.dependsOn}-${phase.id}`,
          source: phase.dependsOn,
          target: phase.id
        });
      }
    });
  });

  return { nodes, edges };
}
