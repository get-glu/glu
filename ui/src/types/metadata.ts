export type Metadata = {
  name: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
};

export const ANNOTATION_OCI_IMAGE_URL = 'dev.getglu.oci.image.url';
export const ANNOTATION_GIT_PROPOSAL_URL = 'dev.getglu.git.proposal.url';
