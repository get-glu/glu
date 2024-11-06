import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import {
  NodeChange,
  EdgeChange,
  Connection,
  applyEdgeChanges,
  applyNodeChanges
} from '@xyflow/react';
import { PipelineNode, PipelineEdge } from '@/types/flow';

interface FlowState {
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

const initialState: FlowState = {
  nodes: [],
  edges: []
};

const flowSlice = createSlice({
  name: 'flow',
  initialState,
  reducers: {
    setFlow: (state, action: PayloadAction<{ nodes: PipelineNode[]; edges: PipelineEdge[] }>) => {
      state.nodes = action.payload.nodes;
      state.edges = action.payload.edges;
    },
    updateNodes: (state, action: PayloadAction<NodeChange[]>) => {
      state.nodes = applyNodeChanges(action.payload, state.nodes) as PipelineNode[];
    },
    updateEdges: (state, action: PayloadAction<EdgeChange[]>) => {
      state.edges = applyEdgeChanges(action.payload, state.edges) as PipelineEdge[];
    },
    addEdge: (state, action: PayloadAction<Connection>) => {
      const newEdge: PipelineEdge = {
        id: `edge-${action.payload.source}-${action.payload.target}`,
        source: action.payload.source,
        target: action.payload.target
      };
      state.edges = [...state.edges, newEdge];
    }
  }
});

export const { setFlow, updateNodes, updateEdges, addEdge } = flowSlice.actions;
export default flowSlice.reducer;
