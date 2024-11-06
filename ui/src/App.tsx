import Workflow from './app/workflow';
import { ThemeProvider } from './components/theme-provider';

export function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="ui-theme">
      <Workflow />
    </ThemeProvider>
  );
}
