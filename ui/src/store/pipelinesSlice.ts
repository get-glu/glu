import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { listPipelines } from '@/services/api';
import { RootState } from '.';
import { Pipeline } from '@/types/pipeline';

export const fetchPipelines = createAsyncThunk('pipelines/fetchPipelines', async () => {
  return await listPipelines();
});

interface PipelinesState {
  data: Pipeline[] | null;
  loading: boolean;
  error: string | null;
  selectedPipeline: Pipeline | null;
}

const initialState: PipelinesState = {
  data: null,
  loading: false,
  error: null,
  selectedPipeline: null
};

export const pipelinesSlice = createSlice({
  name: 'pipelines',
  initialState,
  reducers: {
    setSelectedPipeline: (state, action: PayloadAction<Pipeline | null>) => {
      state.selectedPipeline = action.payload;
    }
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchPipelines.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchPipelines.fulfilled, (state, action) => {
        state.loading = false;
        state.data = action.payload;
      })
      .addCase(fetchPipelines.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch pipelines';
      });
  }
});

export const selectPipelineByName = (state: RootState, name?: string): Pipeline | undefined => {
  if (!name) return undefined;
  return state.pipelines.data?.find((pipeline: Pipeline) => pipeline.name === name);
};

export const selectSelectedPipeline = (state: RootState): Pipeline | null =>
  state.pipelines.selectedPipeline;

export const { setSelectedPipeline } = pipelinesSlice.actions;

export default pipelinesSlice.reducer;

export type { PipelinesState };
