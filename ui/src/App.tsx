import Layout from './app/layout';
import Root from './app/root';
import { ThemeProvider } from './components/theme-provider';
import { createHashRouter, RouterProvider } from 'react-router-dom';
import Pipeline from './app/pipeline';
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
  let title = 'Glu'; // TODO: get system name from backend

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
