import System from './app/system';
import { ThemeProvider } from './components/theme-provider';

export function App() {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="ui-theme">
      <System key={'system'} />
    </ThemeProvider>
  );
}
