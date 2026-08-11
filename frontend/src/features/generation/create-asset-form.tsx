import { LoaderCircle } from "lucide-react";
import { useForm, useStore } from "@tanstack/react-form";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { getAssetKindConfig } from "@/components/asset-kind";
import type { CreatableAssetKind } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import type { ProjectSummary } from "@/model/project";
import {
  assetCreationDraftSchema,
  createAssetCreationDraft,
  toCreationRequest,
} from "./lib";
import type { AssetCreationDraft } from "./types";
import { TilesetAssetFields } from "./create-asset/tileset-asset-fields";
import { VisualAssetFields } from "./create-asset/visual-asset-fields";
import { SceneryAssetFields } from "./create-asset/scenery-asset-fields";
import { UISetAssetFields } from "./create-asset/uiset-asset-fields";

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
  const { t } = useTranslation("workspace");
  const [useProjectContext, setUseProjectContext] = useState(true);
  const [validationError, setValidationError] = useState<string>();
  const form = useForm({
    defaultValues: { draft: createAssetCreationDraft<File>(kind) },
    onSubmit: async ({ value }) => {
      const submittedDraft = { ...value.draft, useProjectContext };
      const result = assetCreationDraftSchema.safeParse(submittedDraft);
      if (!result.success) {
        setValidationError(
          [...new Set(result.error.issues.map((issue) => issue.message))].join(
            " ",
          ),
        );
        return;
      }

      setValidationError(undefined);
      await onCreate(toCreationRequest(submittedDraft));
    },
  });
  const draft = useStore(form.store, (state) => state.values.draft);
  const setDraft = (nextDraft: AssetCreationDraft<File>) => {
    setValidationError(undefined);
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
          {t("generation.assetName")}
          <Input
            required
            placeholder={
              draft.kind === "audio"
                ? t("generation.audioNamePlaceholder")
                : draft.kind === "character"
                  ? t("generation.characterNamePlaceholder")
                  : t("generation.objectNamePlaceholder")
            }
            value={draft.name}
            onChange={(event) =>
              setDraft({ ...draft, name: event.target.value })
            }
          />
        </label>
        <label className="grid gap-2 text-sm font-medium lg:col-span-2">
          {t("generation.creativeBrief")}
          <Textarea
            required
            className="min-h-28 resize-none"
            placeholder={
              draft.kind === "audio"
                ? t("generation.audioPromptPlaceholder")
                : t("generation.promptPlaceholder")
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
        </>
      ) : draft.kind === "uiset" ? (
        <UISetAssetFields draft={draft} onChange={setDraft} />
      ) : draft.kind === "character" || draft.kind === "object" ? (
        <VisualAssetFields draft={draft} onChange={setDraft} />
      ) : null}

      {validationError ? (
        <p className="text-sm text-destructive" role="alert">
          {validationError}
        </p>
      ) : null}

      <label className="flex items-center gap-2 text-sm text-muted-foreground">
        <input
          type="checkbox"
          className="size-4 accent-primary"
          checked={useProjectContext}
          onChange={(event) => setUseProjectContext(event.target.checked)}
        />
        {t("generation.useContext", { name: project.name })}
      </label>

      {useProjectContext ? (
        <div className="border bg-muted/40 p-4">
          <p className="text-xs font-medium text-muted-foreground">
            {t("generation.context")}
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
          {error.message || t("generation.createError")}
        </p>
      ) : null}

      <div className="flex flex-col-reverse gap-2 border-t pt-5 sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          disabled={isSubmitting}
          onClick={onCancel}
        >
          {t("actions.cancel")}
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <LoaderCircle className="animate-spin" /> : null}
          {isSubmitting
            ? t("actions.creating")
            : t("actions.create") + ` ${getAssetKindConfig(kind).label}`}
        </Button>
      </div>
    </form>
  );
}
