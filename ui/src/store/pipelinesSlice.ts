import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { RootState } from '.';

interface PipelinesState {
  selectedPipeline: string;
}

const initialState: PipelinesState = {
  selectedPipeline: ''
};

export const pipelinesSlice = createSlice({
  name: 'pipelines',
  initialState,
  reducers: {
    setSelectedPipeline: (state, action: PayloadAction<string>) => {
      state.selectedPipeline = action.payload;
    }
  }
});

export const getSelectedPipeline = (state: RootState): string => state.pipelines.selectedPipeline;

export const { setSelectedPipeline } = pipelinesSlice.actions;

export default pipelinesSlice.reducer;

export type { PipelinesState };
