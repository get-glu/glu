import axios from 'axios';
import { Pipeline } from '@/types/pipeline';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1'
});

export const getPipeline = async (name: string): Promise<Pipeline> => {
  const response = await api.get<Pipeline>(`/pipelines/${name}`);
  return response.data;
};
