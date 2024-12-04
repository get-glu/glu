import { configureStore } from '@reduxjs/toolkit';
import { api } from '@/services/api';
import pipelinesReducer from './pipelinesSlice';

export const store = configureStore({
  reducer: {
    [api.reducerPath]: api.reducer,
    pipelines: pipelinesReducer
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware)
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
