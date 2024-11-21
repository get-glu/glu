import '@xyflow/react/dist/style.css';
import { Pipeline as PipelineComponent } from '@/components/pipeline';
import { ReactFlowProvider } from '@xyflow/react';
import { useParams } from 'react-router-dom';
import { useAppSelector } from '@/store/hooks';
import { selectPipelineByName, setSelectedPipeline } from '@/store/pipelinesSlice';
import { RootState } from '@/store';
import { useEffect } from 'react';
import { useAppDispatch } from '@/store/hooks';

export default function Pipeline() {
  const { pipelineId } = useParams();
  const dispatch = useAppDispatch();

  const { data: pipelines, loading } = useAppSelector((state: RootState) => state.pipelines);
  const pipeline = useAppSelector((state: RootState) => selectPipelineByName(state, pipelineId));

  useEffect(() => {
    if (pipeline) {
      dispatch(setSelectedPipeline(pipeline));
    }
    return () => {
      dispatch(setSelectedPipeline(null));
    };
  }, [pipeline, dispatch]);

  if (loading || !pipelines) {
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
