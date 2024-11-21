import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { RootState } from '.';
import { Pipeline } from '@/types/pipeline';

interface PipelinesState {
  selectedPipeline: Pipeline | null;
}

const initialState: PipelinesState = {
  selectedPipeline: null
};

export const pipelinesSlice = createSlice({
  name: 'pipelines',
  initialState,
  reducers: {
    setSelectedPipeline: (state, action: PayloadAction<Pipeline | null>) => {
      state.selectedPipeline = action.payload;
    }
  }
});

export const selectSelectedPipeline = (state: RootState): Pipeline | null =>
  state.pipelines.selectedPipeline;

export const { setSelectedPipeline } = pipelinesSlice.actions;

export default pipelinesSlice.reducer;

export type { PipelinesState };
