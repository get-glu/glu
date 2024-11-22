import Layout from './app/layout';
import Root from './app/root';
import { ThemeProvider } from './components/theme-provider';
import { createHashRouter, RouterProvider } from 'react-router-dom';
import Pipeline from './app/pipeline';
import { Helmet } from 'react-helmet';
import { useGetSystemQuery } from './services/api';
import { toast } from 'sonner';

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
  const { data: system, isError, error } = useGetSystemQuery();
  const { toast } = useToast();

  useEffect(() => {
    if (isError) {
      toast.error(error.data);
    }
  }, [error, isError]);

  let title = 'Glu';
  if (system) {
    title = `Glu - ${system.name}`;
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
