import { Pipeline } from '@/types/pipeline';
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
    promotePhase: builder.mutation<void, { pipeline: string; phase: string }>({
      query: ({ pipeline, phase }) => ({
        url: `/pipelines/${pipeline}/phases/${phase}/promote`,
        method: 'POST'
      })
    })
  })
});

export const { useGetSystemQuery, useListPipelinesQuery, useGetPipelineQuery, usePromotePhaseMutation } = api;
