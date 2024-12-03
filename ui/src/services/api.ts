import { Phase, Pipeline, Result, State } from '@/types/pipeline';
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
    getPhase: builder.query<Phase, { pipeline: string; phase: string }>({
      query: ({ pipeline, phase }) => `/pipelines/${pipeline}/phases/${phase}`
    }),
    getPhaseHistory: builder.query<State[], { pipeline: string; phase: string }>({
      query: ({ pipeline, phase }) => `/pipelines/${pipeline}/phases/${phase}/history`
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
  useGetPhaseQuery,
  useGetPhaseHistoryQuery,
  useEdgePerformMutation
} = api;
