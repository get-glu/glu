import axios from 'axios';
import { Pipeline } from '@/types/pipeline';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1'
});

export const listPipelines = async (): Promise<Pipeline[]> => {
  const response = await api.get<{ pipelines: Pipeline[] }>('/pipelines');
  return response.data.pipelines;
};
