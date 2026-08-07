import { LoaderCircle } from "lucide-react";
import { useForm, useStore } from "@tanstack/react-form";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { getAssetKindConfig } from "@/components/asset-kind";
import type { CreatableAssetKind } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";
import {
  createAssetCreationDraft,
  toCreationRequest,
  type AssetCreationDraft,
} from "./types";
import { TilesetAssetFields } from "./create-asset/tileset-asset-fields";
import { VisualAssetFields } from "./create-asset/visual-asset-fields";
import { SceneryAssetFields } from "./create-asset/scenery-asset-fields";
import { UiAssetFields } from "./create-asset/ui-asset-fields";

export function CreateAssetForm({
  kind,
  onCancel,
  onCreate,
  project,
  error,
  isSubmitting = false,
}: {
  kind: CreatableAssetKind;
  onCancel: () => void;
  onCreate: (request: CreationRequest<File>) => void | Promise<void>;
  project: ProjectSummary;
  error?: Error | null;
  isSubmitting?: boolean;
}) {
  const [useProjectContext, setUseProjectContext] = useState(true);
  const [tilesetShapeError, setTilesetShapeError] = useState(false);
  const form = useForm({
    defaultValues: { draft: createAssetCreationDraft<File>(kind) },
    onSubmit: async ({ value }) => {
      if (
        value.draft.kind === "tileset" &&
        value.draft.tiles.some((tile) => tile.tiles.length === 0)
      ) {
        setTilesetShapeError(true);
        return;
      }

      await onCreate(toCreationRequest({ ...value.draft, useProjectContext }));
    },
  });
  const draft = useStore(form.store, (state) => state.values.draft);
  const setDraft = (nextDraft: AssetCreationDraft<File>) => {
    if (
      nextDraft.kind === "tileset" &&
      nextDraft.tiles.every((tile) => tile.tiles.length > 0)
    ) {
      setTilesetShapeError(false);
    }
    form.setFieldValue("draft", nextDraft);
  };

  return (
    <form
      className="grid gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <div className="grid gap-4 lg:grid-cols-2">
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
        <label className="grid gap-2 text-sm font-medium lg:col-span-2">
          Creative brief
          <Textarea
            required
            className="min-h-28 resize-none"
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
      </div>

      {draft.kind === "scenery" ? (
        <SceneryAssetFields draft={draft} onChange={setDraft} />
      ) : draft.kind === "tileset" ? (
        <>
          <TilesetAssetFields draft={draft} onChange={setDraft} />
          {tilesetShapeError ? (
            <p className="text-sm text-destructive" role="alert">
              Each tileset item must have at least one occupied tile.
            </p>
          ) : null}
        </>
      ) : draft.kind === "ui" ? (
        <UiAssetFields draft={draft} onChange={setDraft} />
      ) : draft.kind === "character" || draft.kind === "object" ? (
        <VisualAssetFields draft={draft} onChange={setDraft} />
      ) : null}

      <label className="flex items-center gap-2 text-sm text-muted-foreground">
        <input
          type="checkbox"
          className="size-4 accent-primary"
          checked={useProjectContext}
          onChange={(event) => setUseProjectContext(event.target.checked)}
        />
        Use {project.name} project context
      </label>

      {useProjectContext ? (
        <div className="border bg-muted/40 p-4">
          <p className="text-xs font-medium text-muted-foreground">
            Generation context
          </p>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {[project.gameType, project.style, project.platform]
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

      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error.message || "Unable to create the asset. Please try again."}
        </p>
      ) : null}

      <div className="flex flex-col-reverse gap-2 border-t pt-5 sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          disabled={isSubmitting}
          onClick={onCancel}
        >
          Cancel
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <LoaderCircle className="animate-spin" /> : null}
          {isSubmitting
            ? "Creating..."
            : `Create ${getAssetKindConfig(kind).label}`}
        </Button>
      </div>
    </form>
  );
}
