import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import CodeMirror from '@uiw/react-codemirror';
import { json as jsonLang } from '@codemirror/lang-json';
import { State } from '@/types/pipeline';
import { Label } from '@/components/label';
import { Button } from '@/components/ui/button';
import { PhaseRollbackDialog } from './phase-rollback-dialog';
import { useState } from 'react';
import { ANNOTATION_GIT_COMMIT_URL } from '@/types/metadata';
import { Github } from 'lucide-react';

interface PhaseStateDetailsProps {
  isOpen: boolean;
  onClose: () => void;
  pipelineId: string;
  phaseId: string;
  state: State;
  latest?: boolean;
}

export function PhaseStateDetails({
  isOpen,
  onClose,
  pipelineId,
  phaseId,
  state,
  latest = false
}: PhaseStateDetailsProps) {
  const json = JSON.stringify(state.resource, null, 2);

  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false);

  return (
    <>
      <Dialog open={isOpen} onOpenChange={onClose}>
        <DialogContent className="flex max-h-[90vh] max-w-4xl flex-col rounded-lg p-10 shadow-lg">
          <DialogHeader className="flex flex-row items-center justify-between border-b border-b-muted pb-4">
            <div className="flex w-full items-center justify-between">
              <DialogTitle className="text-xl font-semibold">{phaseId}</DialogTitle>
              {latest ? (
                <Button variant="outline" className="cursor-default dark:text-primary" size="sm">
                  Current Version
                </Button>
              ) : (
                <Button variant="outline" size="sm" onClick={() => setRollbackDialogOpen(true)}>
                  Rollback To This Version
                </Button>
              )}
            </div>
          </DialogHeader>
          <ScrollArea className="mt-2 flex-grow">
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="text-sm">
                  <span className="text-foreground">Recorded At: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 text-xs">
                    {new Date(state.recorded_at).toUTCString()}
                  </span>
                </div>
                <div className="text-sm">
                  <span className="text-foreground">Digest: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 font-mono text-xs">
                    {state.digest}
                  </span>
                </div>
                <div className="text-sm">
                  <span className="text-foreground">Version: </span>
                  <span className="inline-flex items-center truncate rounded-md bg-muted px-2 py-1 font-mono text-xs">
                    {state.version}
                  </span>
                </div>
              </div>

              <div className="space-y-4">
                <h3 className="text-lg font-semibold">Links</h3>
                <div className="flex flex-row flex-wrap gap-2">
                  {state.annotations?.[ANNOTATION_GIT_COMMIT_URL]?.startsWith(
                    'https://github.com'
                  ) && (
                    <a
                      href={state.annotations[ANNOTATION_GIT_COMMIT_URL]}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2 py-1 text-sm hover:bg-muted/80"
                    >
                      <Github className="h-3.5 w-3.5" />
                      Commit
                    </a>
                  )}
                </div>
              </div>

              <div className="space-y-4">
                <h3 className="text-lg font-semibold">Annotations</h3>
                <div className="flex flex-row flex-wrap gap-2">
                  {Object.entries(state.annotations ?? {}).map(([key, value]) => (
                    <span key={`${key}-${value}`}>
                      <Label labelKey={key} value={value} />
                    </span>
                  ))}
                </div>
              </div>

              <div className="space-y-4">
                <h3 className="text-lg font-semibold">Resource</h3>
                <div className="overflow-hidden rounded-md border border-gray-300">
                  <CodeMirror
                    value={json}
                    height="200px"
                    extensions={[jsonLang()]}
                    editable={false}
                    theme="dark"
                  />
                </div>
              </div>
            </div>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      <PhaseRollbackDialog
        isOpen={rollbackDialogOpen}
        onClose={() => setRollbackDialogOpen(false)}
        pipelineId={pipelineId}
        phaseId={phaseId}
        state={state}
      />
    </>
  );
}
