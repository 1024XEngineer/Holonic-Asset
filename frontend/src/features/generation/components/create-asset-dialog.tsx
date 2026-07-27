import { useForm } from "@tanstack/react-form";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { getAssetTypeConfig } from "@/features/assets/components";
import type { CreatableAssetKind } from "@/features/assets/domain";
import type { CreationRequest } from "@/features/generation/domain";
import type { ProjectSummary } from "@/features/project";
import {
  createAssetCreationDraft,
  toCreationRequest,
  type AssetCreationDraft,
} from "@/features/generation/domain/asset-creation";

import {
  SceneryAssetFields,
  TilesetAssetFields,
} from "./CreateAsset/BackgroundAssetFields";
import { UiAssetFields } from "./CreateAsset/UiAssetFields";
import { VisualAssetFields } from "./CreateAsset/VisualAssetFields";

export function CreateAssetDialog({
  children,
  initialPrompt = "",
  onCreate,
  project,
}: {
  children: (openDialog: (kind: CreatableAssetKind) => void) => React.ReactNode;
  initialPrompt?: string;
  onCreate: (request: CreationRequest) => void;
  project: ProjectSummary;
}) {
  const [open, setOpen] = useState(false);
  const form = useForm({
    defaultValues: { draft: createAssetCreationDraft<File>("character") },
    onSubmit: ({ value }) => {
      onCreate(toCreationRequest(value.draft));
      setOpen(false);
    },
  });
  const draft = form.state.values.draft;
  const setDraft = (nextDraft: AssetCreationDraft<File>) =>
    form.setFieldValue("draft", nextDraft);

  const openDialog = (kind: CreatableAssetKind) => {
    form.reset({ draft: createAssetCreationDraft<File>(kind, initialPrompt) });
    setOpen(true);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children(openDialog)}
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            Create {getAssetTypeConfig(draft.kind).label}
          </DialogTitle>
          <DialogDescription>
            Set the production details for this{" "}
            {getAssetTypeConfig(draft.kind).label.toLowerCase()}. Project
            defaults will guide its{" "}
            {draft.kind === "audio" ? "tone and atmosphere" : "visual style"}.
          </DialogDescription>
        </DialogHeader>

        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        >
          <label className="grid gap-2 text-sm font-medium">
            Asset name
            <Input
              required
              placeholder={
                draft.kind === "audio"
                  ? "e.g. Orchard at Night"
                  : `e.g. ${draft.kind === "character" ? "Orchard Keeper" : "Moonlit Lantern"}`
              }
              value={draft.name}
              onChange={(event) =>
                setDraft({ ...draft, name: event.target.value })
              }
            />
          </label>

          <label className="grid gap-2 text-sm font-medium">
            Creative brief
            <Textarea
              required
              className="min-h-24 resize-y"
              placeholder={
                draft.kind === "audio"
                  ? "Describe the mood, instruments, rhythm, and intended use..."
                  : "Describe the subject, material, mood, and details to generate..."
              }
              value={draft.prompt}
              onChange={(event) =>
                setDraft({ ...draft, prompt: event.target.value })
              }
            />
          </label>

          {draft.kind === "scenery" ? (
            <SceneryAssetFields draft={draft} onChange={setDraft} />
          ) : draft.kind === "tileset" ? (
            <TilesetAssetFields draft={draft} onChange={setDraft} />
          ) : draft.kind === "ui" ? (
            <UiAssetFields draft={draft} onChange={setDraft} />
          ) : draft.kind === "audio" ? null : (
            <VisualAssetFields draft={draft} onChange={setDraft} />
          )}

          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <Checkbox
              checked={draft.useProjectContext}
              onCheckedChange={(useProjectContext) =>
                setDraft({ ...draft, useProjectContext })
              }
            />
            Use {project.name} project context
          </label>

          {draft.useProjectContext ? (
            <div className="rounded-lg border bg-muted/40 p-3">
              <p className="text-xs font-medium text-muted-foreground">
                Generation context
              </p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {[project.gameType, project.visualStyle, project.platform]
                  .filter(Boolean)
                  .map((item) => (
                    <Badge key={item} variant="secondary">
                      {item}
                    </Badge>
                  ))}
              </div>
              {project.description ? (
                <p className="mt-2 line-clamp-2 text-xs leading-5 text-muted-foreground">
                  {project.description}
                </p>
              ) : null}
            </div>
          ) : null}

          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>
              Cancel
            </DialogClose>
            <Button type="submit">
              Create {getAssetTypeConfig(draft.kind).label}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
