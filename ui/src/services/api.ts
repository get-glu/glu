import { Pipeline, Result } from '@/types/pipeline';
import { System } from '@/types/system';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export const api = createApi({
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  endpoints: (builder) => ({
    getSystem: builder.query<System, void>({
      query: () => '/'
    }),
    listPipelines: builder.query<{ pipelines: Pipeline[] }, void>({
      query: () => '/pipelines'
    }),
    getPipeline: builder.query<Pipeline, string>({
      query: (pipeline) => `/pipelines/${pipeline}`
    }),
    edgePerform: builder.mutation<Result, { pipeline: string; from: string; to: string }>({
      query: ({ pipeline, from, to }) => ({
        url: `/pipelines/${pipeline}/from/${from}/to/${to}/perform`,
        method: 'POST'
      })
    })
  })
});

export const {
  useGetSystemQuery,
  useListPipelinesQuery,
  useGetPipelineQuery,
  useEdgePerformMutation
} = api;
