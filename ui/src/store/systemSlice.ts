import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import { getSystem } from '@/services/api';

export const fetchSystem = createAsyncThunk('system/fetchSystem', async () => {
  return await getSystem();
});

interface SystemType {
  name: string;
  labels?: Record<string, string>;
  // add other system properties you need
}

interface SystemState {
  data: SystemType | null;
  loading: boolean;
  error: string | null;
}

const initialState: SystemState = {
  data: null,
  loading: false,
  error: null
};

export const systemSlice = createSlice({
  name: 'system',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchSystem.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchSystem.fulfilled, (state, action) => {
        state.loading = false;
        state.data = action.payload;
      })
      .addCase(fetchSystem.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch system';
      });
  }
});

export default systemSlice.reducer;

export type { SystemType, SystemState };
