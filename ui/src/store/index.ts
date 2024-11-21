import { configureStore } from '@reduxjs/toolkit';
import { systemSlice, SystemState } from './systemSlice';
import { pipelinesSlice, PipelinesState } from './pipelinesSlice';

export interface StoreState {
  system: SystemState;
  pipelines: PipelinesState;
}

export const store = configureStore({
  reducer: {
    system: systemSlice.reducer,
    pipelines: pipelinesSlice.reducer
  }
});

export type RootState = StoreState;
export type AppDispatch = typeof store.dispatch;
