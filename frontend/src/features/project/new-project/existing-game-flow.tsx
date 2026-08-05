import { useRef } from "react";
import { Gamepad2, Link2, Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { NewProjectController } from "./use-new-project-controller";
import { GuidedProjectFlow } from "./guided-project-flow";
import { ProjectStartCard } from "./project-start-card";

export function ExistingGameFlow({
  active,
  project,
}: {
  active: boolean;
  project: NewProjectController;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { existingGameImport } = project;

  if (active) return <GuidedProjectFlow project={project} />;

  return (
    <>
      <ProjectStartCard
        title="Existing game"
        description="Import a playable link or upload a local build so we can learn its direction."
        icon={<Gamepad2 size={20} />}
        onSelect={project.start.openExistingGameImport}
      />
      <Dialog
        open={existingGameImport.isOpen}
        onOpenChange={(open) => !open && existingGameImport.dismiss()}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Import an existing game</DialogTitle>
            <DialogDescription>
              Share a playable link or upload a local game build to begin its
              project setup.
            </DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-2 rounded-lg bg-muted p-1">
            <Button
              variant={existingGameImport.mode === "link" ? "default" : "ghost"}
              onClick={existingGameImport.selectLink}
            >
              <Link2 /> Game link
            </Button>
            <Button
              variant={existingGameImport.mode === "file" ? "default" : "ghost"}
              onClick={existingGameImport.selectFile}
            >
              <Upload /> Local files
            </Button>
          </div>
          {existingGameImport.mode === "link" ? (
            <label className="grid gap-2 text-sm font-medium">
              Playable URL
              <Input
                type="url"
                placeholder="https://your-game.example"
                value={existingGameImport.gameUrl}
                onChange={(event) =>
                  existingGameImport.setGameUrl(event.target.value)
                }
              />
            </label>
          ) : (
            <>
              <input
                ref={fileInputRef}
                className="sr-only"
                type="file"
                accept=".zip,.html,.exe,.dmg,.apk"
                onChange={(event) =>
                  existingGameImport.setGameFile(
                    event.target.files?.[0] ?? null,
                  )
                }
              />
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                className="grid min-h-32 place-items-center rounded-xl border border-dashed p-5 text-center text-sm hover:bg-muted/50"
              >
                <span>
                  <Upload className="mx-auto mb-2 size-5 text-muted-foreground" />
                  {existingGameImport.gameFile?.name ?? "Choose a game build"}
                </span>
              </button>
            </>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={existingGameImport.dismiss}>
              Cancel
            </Button>
            <Button
              disabled={
                existingGameImport.mode === "link"
                  ? !existingGameImport.gameUrl.trim()
                  : !existingGameImport.gameFile
              }
              onClick={existingGameImport.continue}
            >
              Continue
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
