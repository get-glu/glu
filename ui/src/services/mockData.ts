import { Pipeline } from '@/types/pipeline';

export const getMockPipelines = async (): Promise<Pipeline[]> => {
  return [
    {
      id: 'cloud-controller',
      name: 'cloud-controller',
      phases: [
        {
          id: 'cloud-controller-oci',
          name: 'OCI',
          type: 'oci',
          pipelineId: 'cloud-controller'
        },
        {
          id: 'cloud-controller-staging',
          name: 'Staging',
          type: 'staging',
          dependsOn: 'cloud-controller-oci',
          pipelineId: 'cloud-controller'
        },
        {
          id: 'cloud-controller-production',
          name: 'Production',
          type: 'production',
          dependsOn: 'cloud-controller-staging',
          pipelineId: 'cloud-controller'
        }
      ]
    },
    {
      id: 'frontdoor',
      name: 'frontdoor',
      phases: [
        {
          id: 'frontdoor-oci',
          name: 'OCI',
          type: 'oci',
          pipelineId: 'frontdoor'
        },
        {
          id: 'frontdoor-staging',
          name: 'Staging',
          type: 'staging',
          dependsOn: 'frontdoor-oci',
          pipelineId: 'frontdoor'
        },
        {
          id: 'frontdoor-production',
          name: 'Production',
          type: 'production',
          dependsOn: 'frontdoor-staging',
          pipelineId: 'frontdoor'
        }
      ]
    }
  ];
};
