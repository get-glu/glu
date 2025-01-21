import { useEffect, useState } from 'react';
import {
  Node,
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
import { PhasePanel } from '@/components/phase-panel';
import { PhaseNode as PhaseNodeComponent } from '@/components/phase-node';
import { Pipeline as PipelineType } from '@/types/pipeline';
import { FlowPipeline, PipelineEdge, PhaseNode } from '@/types/flow';
import Dagre from '@dagrejs/dagre';
import { useGetPipelineQuery } from '@/services/api';

const nodeTypes = {
  phase: PhaseNodeComponent
};

export function Pipeline({ pipelineId }: { pipelineId: string }) {
  const { theme } = useTheme();
  const { fitView } = useReactFlow<PhaseNode, PipelineEdge>();
  const [selectedNode, setSelectedNode] = useState<PhaseNode | null>(null);
  const [isPanelExpanded, setIsPanelExpanded] = useState(true);

  const { data: pipeline } = useGetPipelineQuery(pipelineId);

  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node[]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as PipelineEdge[]);

  const nodesInitialized = useNodesInitialized();

  useEffect(() => {
    if (pipeline) {
      const { nodes: newNodes, edges: newEdges } = getElements(pipeline);
      const flow = getLayoutedElements({ nodes: newNodes, edges: newEdges });

      setNodes(flow.nodes);
      setEdges(flow.edges);
    }
  }, [pipeline]);

  useEffect(() => {
    if (nodesInitialized) {
      const flow = getLayoutedElements({ nodes, edges });
      setNodes(flow.nodes);
      setEdges(flow.edges);
    }
  }, [nodesInitialized]);

  useEffect(() => {
    fitView();
    const handleResize = () => fitView();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [nodes, edges, fitView]);

  const onNodeClick = (_event: React.MouseEvent, node: Node) => {
    setSelectedNode(node as PhaseNode);
  };

  return (
    <div className="flex h-screen w-full flex-col">
      <div className="min-h-0 flex-1">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={onNodeClick}
          fitView
          className="h-full"
          colorMode={theme}
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{
            markerEnd: {
              type: MarkerType.Arrow,
              width: 20,
              height: 20,
              color: 'currentColor'
            },
            selectable: true,
            style: {
              strokeWidth: 2
            }
          }}
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>
      {selectedNode && (
        <PhasePanel
          node={selectedNode}
          isExpanded={isPanelExpanded}
          onToggle={() => setIsPanelExpanded(!isPanelExpanded)}
        />
      )}
    </div>
  );
}

function getLayoutedElements(pipeline: FlowPipeline) {
  const g = new Dagre.graphlib.Graph({ compound: true }).setDefaultEdgeLabel(() => ({}));

  // Set larger node separation and rank separation
  g.setGraph({
    rankdir: 'LR',
    nodesep: 10, // Horizontal spacing between nodes in the same rank
    ranksep: 100, // Spacing between ranks (vertical layers)
    marginx: 10, // Margin from left/right edges
    marginy: 10 // Margin from top/bottom edges
  });

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

function getElements(pipeline: PipelineType): FlowPipeline {
  const nodes: PhaseNode[] = [];
  const edges: PipelineEdge[] = [];

  pipeline.phases.forEach((phase) => {
    const node: PhaseNode = {
      id: phase.descriptor.metadata.name,
      type: 'phase',
      position: { x: 0, y: 0 },
      data: {
        descriptor: phase.descriptor,
        status: phase.status
      },
      extent: 'parent'
    };
    nodes.push(node);
  });

  pipeline.edges.forEach((e) => {
    const edge: PipelineEdge = {
      id: `edge-${e.from.metadata.name}-${e.to.metadata.name}`,
      source: e.from.metadata.name,
      target: e.to.metadata.name,
      data: {
        kind: e.kind,
        from: e.from,
        to: e.to
      }
    };

    edges.push(edge);
  });

  return { nodes, edges };
}
