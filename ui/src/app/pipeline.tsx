import '@xyflow/react/dist/style.css';
import { Pipeline as PipelineComponent } from '@/components/pipeline';
import { ReactFlowProvider } from '@xyflow/react';
import { useParams } from 'react-router-dom';
import { setSelectedPipeline } from '@/store/pipelinesSlice';
import { useAppDispatch } from '@/store/hooks';

export default function Pipeline() {
  const { pipelineId } = useParams();
  const dispatch = useAppDispatch();

  if (!pipelineId) {
    return <div>Pipeline not found: {pipelineId}</div>;
  }

  dispatch(setSelectedPipeline(pipelineId));

  return (
    <ReactFlowProvider key={`provider-${pipelineId}`}>
      <PipelineComponent pipelineId={pipelineId} />
    </ReactFlowProvider>
  );
}
