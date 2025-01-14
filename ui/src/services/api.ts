import { Phase, Pipeline } from '@/types/pipeline';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export const api = createApi({
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  endpoints: (builder) => ({
    // getSystem: builder.query<System, void>({
    //   query: () => '/'
    // }),
    listPipelines: builder.query<{ pipelines: Pipeline[] }, void>({
      query: () => '/pipelines'
    }),
    getPipeline: builder.query<Pipeline, string>({
      query: (pipeline) => `/pipelines/${pipeline}`
    }),
    getPhase: builder.query<Phase, { pipeline: string; phase: string }>({
      query: ({ pipeline, phase }) => `/pipelines/${pipeline}/phases/${phase}`
    })
  })
});

export const { useListPipelinesQuery, useGetPipelineQuery, useGetPhaseQuery } = api;
