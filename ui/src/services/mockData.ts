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
          pipelineId: 'cloud-controller',
          labels: {
            version: 'v1.2.3',
            status: 'running'
          }
        },
        {
          id: 'cloud-controller-staging',
          name: 'Staging',
          type: 'staging',
          dependsOn: 'cloud-controller-oci',
          pipelineId: 'cloud-controller',
          labels: {
            environment: 'staging',
            region: 'us-east-1'
          }
        },
        {
          id: 'cloud-controller-production-east-1',
          name: 'Production (East 1)',
          type: 'production',
          dependsOn: 'cloud-controller-staging',
          pipelineId: 'cloud-controller',
          labels: {
            environment: 'production',
            region: 'us-east-1',
            replicas: '3'
          }
        },
        {
          id: 'cloud-controller-production-west-1',
          name: 'Production (West 1)',
          type: 'production',
          dependsOn: 'cloud-controller-staging',
          pipelineId: 'cloud-controller',
          labels: {
            environment: 'production',
            region: 'us-west-1',
            replicas: '3'
          }
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
          pipelineId: 'frontdoor',
          labels: {
            version: 'v2.0.1',
            builder: 'docker'
          }
        },
        {
          id: 'frontdoor-staging',
          name: 'Staging',
          type: 'staging',
          dependsOn: 'frontdoor-oci',
          pipelineId: 'frontdoor',
          labels: {
            environment: 'staging',
            domain: 'stage.example.com'
          }
        },
        {
          id: 'frontdoor-production',
          name: 'Production',
          type: 'production',
          dependsOn: 'frontdoor-staging',
          pipelineId: 'frontdoor',
          labels: {
            environment: 'production',
            domain: 'prod.example.com',
            ssl: 'enabled'
          }
        }
      ]
    }
  ];
};
