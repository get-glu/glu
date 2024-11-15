import { useEffect, useState } from 'react';
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
import '@xyflow/react/dist/style.css';
import { useTheme } from '@/components/theme-provider';
import { PhaseNode as PhaseNodeComponent } from '@/components/node';
import { Pipeline } from '@/types/pipeline';
import { FlowPipeline, PipelineEdge, PhaseNode, PipelineNode } from '@/types/flow';
import Dagre from '@dagrejs/dagre';

const nodeTypes = {
  phase: PhaseNodeComponent
};

export function Pipeline(props: { pipeline: Pipeline }) {
  const { theme } = useTheme();
  const { fitView } = useReactFlow<PipelineNode, PipelineEdge>();

  const { pipeline } = props;
  const { nodes: initNodes, edges: initEdges } = getElements(pipeline);

  const [nodes, setNodes, onNodesChange] = useNodesState(initNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initEdges);

  const nodesInitialized = useNodesInitialized();
  const [initLayoutFinished, setInitLayoutFinished] = useState(false);

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
    <div key={`pipeline-${pipeline.name}`} className="mb-10 flex h-[500px] w-full flex-col">
      <div className="mb-5 mt-5 flex bg-background px-2 text-lg font-medium text-muted-foreground">
        {pipeline.name}
      </div>
      <div className="mb-5 flex h-full w-full border-2 border-solid border-black">
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

function getElements(pipeline: Pipeline): FlowPipeline {
  const nodes: PipelineNode[] = [];
  const edges: PipelineEdge[] = [];

  pipeline.phases.forEach((phase) => {
    const node: PhaseNode = {
      id: phase.name,
      type: 'phase',
      position: { x: 0, y: 0 },
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

  return { nodes, edges };
}
