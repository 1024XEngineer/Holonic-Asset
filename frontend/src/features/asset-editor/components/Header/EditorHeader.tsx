import { ArrowLeft, Redo2, Save, Undo2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

type EditorHeaderProps = {
  assetName: string;
  version: string;
  projectName: string;
  onBack: () => void;
  status: string;
  canUndo: boolean;
  canRedo: boolean;
  isSaving: boolean;
  onUndo: () => void;
  onRedo: () => void;
  onSave: () => void;
};

export function EditorHeader({
  assetName,
  version,
  projectName,
  onBack,
  status,
  canUndo,
  canRedo,
  isSaving,
  onUndo,
  onRedo,
  onSave,
}: EditorHeaderProps) {
  return (
    <header className="flex h-[4.5rem] shrink-0 items-center justify-between border-b border-black/10 bg-[#ffffff] px-4 lg:px-6">
      <div className="flex min-w-0 items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          aria-label={`Back to ${assetName}`}
          className="text-[#776e63] hover:bg-black/5 hover:text-[#2d2923]"
          onClick={onBack}
        >
          <ArrowLeft />
        </Button>
        <div className="hidden h-7 w-px bg-black/5 sm:block" />
        <div className="min-w-0">
          <div className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-[0.18em] text-[#7d7468]">
            <span className="text-[#b86b70]">{projectName}</span>
            <span className="text-black/20">/</span>
            <span>Asset studio</span>
          </div>
          <div className="mt-1 flex items-center gap-2">
            <h1 className="truncate font-serif text-lg tracking-tight text-[#2d2923] sm:text-xl">
              {assetName}
            </h1>
            <Badge className="hidden border-[#b86b70]/30 bg-[#b86b70]/10 text-[#b86b70] sm:inline-flex">
              {version}
            </Badge>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <span className="hidden items-center gap-2 text-xs text-[#786f64] lg:flex">
          <span className="size-1.5 rounded-full bg-[#579672] shadow-[0_0_0_3px_rgba(152,215,173,.12)]" />
          {status}
        </span>
        <Button
          variant="outline"
          size="icon"
          aria-label="Undo"
          disabled={!canUndo}
          className="border-black/10 bg-black/[.025] text-[#39342d] hover:bg-black/5"
          onClick={onUndo}
        >
          <Undo2 />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label="Redo"
          disabled={!canRedo}
          className="border-black/10 bg-black/[.025] text-[#39342d] hover:bg-black/5"
          onClick={onRedo}
        >
          <Redo2 />
        </Button>
        <Button
          size="sm"
          disabled={isSaving}
          className="bg-[#2d2923] text-white hover:bg-[#4b453d]"
          onClick={onSave}
        >
          <Save data-icon="inline-start" />
          {isSaving ? "Saving" : "Save"}
        </Button>
      </div>
    </header>
  );
}
