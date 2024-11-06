import { configureStore } from '@reduxjs/toolkit';
import pipelineReducer from './pipelineSlice';
import flowReducer from './flowSlice';

export const store = configureStore({
  reducer: {
    pipeline: pipelineReducer,
    flow: flowReducer
  }
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
