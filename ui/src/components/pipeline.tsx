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
import { Pipeline as PipelineType } from '@/types/pipeline';
import { FlowPipeline, PipelineEdge, PhaseNode, PipelineNode } from '@/types/flow';
import Dagre from '@dagrejs/dagre';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ChevronDown } from 'lucide-react';

const nodeTypes = {
  phase: PhaseNodeComponent
};

export function Pipeline(props: { pipeline: PipelineType }) {
  const { theme } = useTheme();
  const { fitView, getNodes, getEdges } = useReactFlow<PipelineNode, PipelineEdge>();
  const [isOpen, setIsOpen] = useState(true);

  const { pipeline } = props;
  const { nodes: initNodes, edges: initEdges } = getElements(pipeline);

  const [nodes, setNodes, onNodesChange] = useNodesState(initNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initEdges);

  const nodesInitialized = useNodesInitialized();

  useEffect(() => {
    if (nodesInitialized) {
      const flow = getLayoutedElements({ nodes: getNodes(), edges: getEdges() });
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

  return (
    <Collapsible
      key={`pipeline-${pipeline.name}`}
      open={isOpen}
      onOpenChange={setIsOpen}
      className="mb-10 w-full"
    >
      <CollapsibleTrigger className="flex w-full items-center gap-2 py-2">
        <ChevronDown
          className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-0' : '-rotate-90'}`}
        />
        <span className="text-lg font-medium">{pipeline.name}</span>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="mb-5 flex h-[500px] w-full border border-solid border-black">
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
      </CollapsibleContent>
    </Collapsible>
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
  const nodes: PipelineNode[] = [];
  const edges: PipelineEdge[] = [];

  pipeline.phases.forEach((phase) => {
    const node: PhaseNode = {
      id: phase.name,
      type: 'phase',
      position: { x: 0, y: 0 },
      data: {
        pipeline: pipeline.name,
        name: phase.name,
        labels: phase.labels || {},
        depends_on: phase.depends_on,
        source: phase.source,
        digest: phase.digest
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
