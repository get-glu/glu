import { ThemeToggle } from './theme-toggle';
import { ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SidebarTrigger } from './ui/sidebar';
import { useAppSelector } from '@/store/hooks';
import { getSelectedPipeline } from '@/store/pipelinesSlice';

export function Header({ className }: { className?: string }) {
  const pipelineId = useAppSelector(getSelectedPipeline);

  return (
    <div className={cn('h-18 border-b bg-background p-4', className)}>
      <div className="flex gap-2">
        <SidebarTrigger className="-ml-1" />
        <div className="flex w-full justify-between">
          <div className="flex items-center gap-2">
            <span className="text-lg font-bold">Glu</span>
            {pipelineId && (
              <>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
                <span className="text-lg text-muted-foreground">{pipelineId}</span>
              </>
            )}
          </div>
          <ThemeToggle />
        </div>
      </div>
    </div>
  );
}
