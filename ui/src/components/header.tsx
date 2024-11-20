import { useAppSelector } from '@/store/hooks';
import { RootState } from '@/store';
import { ThemeToggle } from './theme-toggle';
import { ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SidebarTrigger } from './ui/sidebar';

export function Header({ className }: { className?: string }) {
  const { data: system, loading } = useAppSelector((state: RootState) => state.system);
  const selectedPipeline = useAppSelector((state: RootState) => state.pipelines.selectedPipeline);

  return (
    <div className={cn('h-18 border-b bg-background p-4', className)}>
      <div className="flex gap-2">
        <SidebarTrigger className="-ml-1" />
        <div className="flex w-full justify-between">
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
    </div>
  );
}
