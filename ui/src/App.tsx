import Layout from './app/layout';
import Root from './app/root';
import { ThemeProvider } from './components/theme-provider';
import { useEffect } from 'react';
import { useAppDispatch, useAppSelector } from './store/hooks';
import { fetchPipelines } from './store/pipelinesSlice';
import { fetchSystem } from './store/systemSlice';
import { createHashRouter, RouterProvider } from 'react-router-dom';
import Pipeline from './app/pipeline';
import { RootState } from './store';
import { Helmet } from 'react-helmet';

const router = createHashRouter([
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
  const system = useAppSelector((state: RootState) => state.system.data?.name);

  useEffect(() => {
    dispatch(fetchSystem());
    dispatch(fetchPipelines());
  }, [dispatch]);

  let title = 'Glu';
  if (system) {
    title = `Glu - ${system}`;
  }

  return (
    <>
      <Helmet>
        <meta charSet="utf-8" />
        <title>{title}</title>
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      </Helmet>
      <ThemeProvider defaultTheme="dark" storageKey="ui-theme">
        <RouterProvider router={router} />
      </ThemeProvider>
    </>
  );
}
