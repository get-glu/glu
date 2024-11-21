import { configureStore } from '@reduxjs/toolkit';
import { api } from '@/services/api';
import { Pipeline } from '@/types/pipeline';
import { pipelinesSlice, PipelinesState } from './pipelinesSlice';

export interface StoreState {
  pipelines: PipelinesState;
}

export const store = configureStore({
  reducer: {
    pipelines: pipelinesSlice.reducer,
    [api.reducerPath]: api.reducer
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware)
});

export type RootState = StoreState;
export type AppDispatch = typeof store.dispatch;
