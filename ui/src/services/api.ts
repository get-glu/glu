import axios from 'axios';
import { Pipeline } from '@/types/pipeline';
import { System } from '@/types/system';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1'
});

export const getSystem = async (): Promise<System> => {
  const response = await api.get<System>('/');
  return response.data;
};

export const listPipelines = async (): Promise<Pipeline[]> => {
  const response = await api.get<{ pipelines: Pipeline[] }>('/pipelines');
  return response.data.pipelines;
};

export const promotePhase = async (pipeline: string, phase: string) => {
  const response = await api.post(`/pipelines/${pipeline}/phase/${phase}/promote`);
  if (response.status !== 200) {
    throw new Error(`unexpected status ${response.status}`);
  }
};
