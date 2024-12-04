import '@xyflow/react/dist/style.css';
import { Pipeline as PipelineComponent } from '@/components/pipeline';
import { ReactFlowProvider } from '@xyflow/react';
import { useParams } from 'react-router-dom';
import { Header } from '@/components/header';

export default function Pipeline() {
  const { pipelineId } = useParams();

  if (!pipelineId) {
    return <div>Pipeline not found: {pipelineId}</div>;
  }

  return (
    <>
      <Header className="absolute left-0 right-0 top-0 z-10" pipelineId={pipelineId} />
      <div className="flex h-screen w-full">
        <ReactFlowProvider key={`provider-${pipelineId}`}>
          <PipelineComponent pipelineId={pipelineId} />
        </ReactFlowProvider>
      </div>
    </>
  );
}
