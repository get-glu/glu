import { useRollbackPhaseMutation } from '@/services/api';
import { State } from '@/types/pipeline';
import { useState } from 'react';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from './ui/dialog';
import { Loader2 } from 'lucide-react';
import { Button } from './ui/button';

interface PhaseRollbackDialogProps {
  isOpen: boolean;
  onClose: () => void;
  pipelineId: string;
  phaseId: string;
  state: State;
}

export function PhaseRollbackDialog({
  isOpen,
  onClose,
  pipelineId,
  phaseId,
  state
}: PhaseRollbackDialogProps) {
  const [isPerformingRollback, setIsPerformingRollback] = useState(false);
  const [rollbackPhase] = useRollbackPhaseMutation();

  const performRollback = async () => {
    try {
      setIsPerformingRollback(true);
      await rollbackPhase({
        pipeline: pipelineId,
        phase: phaseId,
        version: state.version
      }).unwrap();

      toast.success('Phase version updated');
    } catch (e) {
      console.error(e);

      toast.error('Something went wrong');
    } finally {
      setIsPerformingRollback(false);
      onClose();
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="px-4">
        <DialogHeader>
          <DialogTitle>Rollback Phase</DialogTitle>
          <DialogDescription className="flex flex-col gap-4">
            Are you sure you want to rollback this phase to this digest?
            <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1">
              <span className="font-mono text-xs">{state.digest}</span>
            </span>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={performRollback}>
            {isPerformingRollback ? (
              <>
                <Loader2 className="animate-spin" /> Please Wait{' '}
              </>
            ) : (
              'Rollback'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
