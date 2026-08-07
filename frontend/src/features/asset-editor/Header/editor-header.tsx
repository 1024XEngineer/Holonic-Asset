import { ArrowLeft, Redo2, Save, Undo2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import {
  GenerationTaskDropdown,
  type EditorGenerationTask,
} from "./generation-task-dropdown";

export type { EditorGenerationTask } from "./generation-task-dropdown";

export function EditorHeader({
  assetKind,
  assetName,
  version,
  projectName,
  onBack,
  status,
  canUndo,
  canRedo,
  isDirty,
  isSaving,
  generationTasks,
  onUndo,
  onRedo,
  onSave,
}: {
  assetKind: "character" | "object";
  assetName: string;
  version: string;
  projectName: string;
  onBack: () => void;
  status: string;
  canUndo: boolean;
  canRedo: boolean;
  isDirty: boolean;
  isSaving: boolean;
  generationTasks: EditorGenerationTask[];
  onUndo: () => void;
  onRedo: () => void;
  onSave: () => void;
}) {
  return (
    <header className="flex min-h-16 shrink-0 items-center justify-between gap-4 border-b bg-background px-3 sm:px-5">
      <div className="flex min-w-0 items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Back to asset library"
          title="Back to asset library"
          onClick={onBack}
        >
          <ArrowLeft />
        </Button>
        <div className="hidden h-7 w-px bg-border sm:block" />
        <div className="min-w-0">
          <div className="hidden items-center gap-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-foreground sm:flex">
            <span>{projectName}</span>
            <span aria-hidden="true">/</span>
            <span>{assetKind} editor</span>
          </div>
          <div className="flex min-w-0 items-center gap-2">
            <h1 className="truncate text-base font-semibold sm:text-lg">
              {assetName}
            </h1>
            <Badge variant="outline" className="hidden sm:inline-flex">
              {version}
            </Badge>
          </div>
        </div>
      </div>

      <div className="hidden min-w-0 flex-1 justify-center px-4 lg:flex">
        <GenerationTaskDropdown tasks={generationTasks} />
      </div>

      <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
        <span className="hidden max-w-36 truncate text-xs text-muted-foreground xl:inline">
          {status}
        </span>
        <Button
          variant="outline"
          size="icon"
          aria-label="Undo"
          title="Undo"
          disabled={!canUndo}
          onClick={onUndo}
        >
          <Undo2 />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Redo"
          title="Redo"
          disabled={!canRedo}
          onClick={onRedo}
        >
          <Redo2 />
        </Button>
        <Button size="sm" disabled={isSaving || !isDirty} onClick={onSave}>
          <Save data-icon="inline-start" />
          <span className="hidden sm:inline">
            {isSaving ? "Saving" : "Save"}
          </span>
        </Button>
      </div>
    </header>
  );
}
