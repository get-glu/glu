import { useAppSelector } from '@/store/hooks';
import { RootState } from '@/store';
import { ThemeToggle } from './theme-toggle';
import { ChevronRight } from 'lucide-react';

export function Header({ className }: HeaderProps) {
  const { data: system, loading } = useAppSelector((state: RootState) => state.system);
  const selectedPipeline = useAppSelector((state: RootState) => state.pipelines.selectedPipeline);

  return (
    <div className="h-18 absolute left-0 right-0 top-0 z-10 border-b bg-background p-4">
      <div className="mx-auto flex max-w-7xl items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold">{loading ? 'Loading...' : system?.name}</span>
          {selectedPipeline && (
            <>
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
              <span className="text-lg text-muted-foreground">{selectedPipeline.name}</span>
            </>
          )}
        </div>
        <ThemeToggle />
      </div>
    </div>
  );
}
