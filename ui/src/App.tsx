import Layout from './app/layout';
import Root from './app/root';
import { ThemeProvider } from './components/theme-provider';
import { useEffect } from 'react';
import { useAppDispatch, useAppSelector } from './store/hooks';
import { fetchPipelines } from './store/pipelinesSlice';
import { fetchSystem } from './store/systemSlice';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import Pipeline from './app/pipeline';
import { RootState } from './store';

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        path: '/',
        element: <Root />
      },
      {
        path: 'pipelines/:pipelineId',
        element: <Pipeline />
      }
    ]
    // errorElement: <ErrorPage />,
  }
]);

export function App() {
  const dispatch = useAppDispatch();
  const pipelinesState = useAppSelector((state: RootState) => state.pipelines);

  useEffect(() => {
    dispatch(fetchSystem());
    dispatch(fetchPipelines());
  }, [dispatch]);

  return (
    <ThemeProvider defaultTheme="dark" storageKey="ui-theme">
      <RouterProvider router={router} />
    </ThemeProvider>
  );
}
