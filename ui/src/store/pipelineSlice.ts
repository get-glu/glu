import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { Pipeline } from '@/types/pipeline';
import { getPipeline } from '@/services/api';

interface PipelineState {
  data: Pipeline | null;
  loading: boolean;
  error: string | null;
}

const initialState: PipelineState = {
  data: null,
  loading: false,
  error: null
};

export const fetchPipeline = createAsyncThunk('pipeline/fetchPipeline', async (name: string) => {
  const response = await getPipeline(name);
  return response;
});

const pipelineSlice = createSlice({
  name: 'pipeline',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchPipeline.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchPipeline.fulfilled, (state, action) => {
        state.loading = false;
        state.data = action.payload;
      })
      .addCase(fetchPipeline.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch pipeline';
      });
  }
});

export default pipelineSlice.reducer;
