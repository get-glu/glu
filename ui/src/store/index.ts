import { configureStore } from '@reduxjs/toolkit';
import { api } from '@/services/api';

export interface StoreState {}

export const store = configureStore({
  reducer: {
    [api.reducerPath]: api.reducer
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(api.middleware)
});

export type RootState = StoreState;
export type AppDispatch = typeof store.dispatch;
