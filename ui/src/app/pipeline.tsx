import '@xyflow/react/dist/style.css';
import { Pipeline as PipelineComponent } from '@/components/pipeline';
import { ReactFlowProvider } from '@xyflow/react';
import { useParams } from 'react-router-dom';
import { setSelectedPipeline } from '@/store/pipelinesSlice';
import { useEffect } from 'react';
import { useAppDispatch } from '@/store/hooks';
import { useGetPipelineQuery } from '@/services/api';

export default function Pipeline() {
  const { pipelineId } = useParams();
  const dispatch = useAppDispatch();

  const { data: pipeline, isLoading } = useGetPipelineQuery(pipelineId ?? '', {
    pollingInterval: 5000
  });

  useEffect(() => {
    if (pipeline) {
      dispatch(setSelectedPipeline(pipeline));
    }
    return () => {
      dispatch(setSelectedPipeline(null));
    };
  }, [pipeline, dispatch]);

  if (isLoading || !pipeline) {
    return <div>Loading pipeline...</div>;
  }

  if (!pipeline) {
    return <div>Pipeline not found: {pipelineId}</div>;
  }

  return (
    <ReactFlowProvider key={`provider-${pipelineId}`}>
      <PipelineComponent key={pipelineId} pipeline={pipeline} />
    </ReactFlowProvider>
  );
}
