import { LoaderCircle } from "lucide-react";
import { useForm, useStore } from "@tanstack/react-form";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { CreatableAssetKind } from "@/model/asset";
import type { CreationRequest } from "@/model/generation";
import {
  assetCreationDraftSchema,
  createAssetCreationDraft,
  toCreationRequest,
} from "./lib";
import type { AssetCreationDraft } from "./types";
import { TilesetAssetFields } from "./create-asset/tileset-asset-fields";
import { VisualAssetFields } from "./create-asset/visual-asset-fields";
import { UISetAssetFields } from "./create-asset/uiset-asset-fields";
import { SceneryAssetFields } from "./create-asset/scenery-asset-fields";

const assetNamePlaceholderKeys = {
  audio: "audioNamePlaceholder",
  character: "characterNamePlaceholder",
  object: "objectNamePlaceholder",
  scenery: "sceneryNamePlaceholder",
  tileset: "tilesetNamePlaceholder",
  uiset: "objectNamePlaceholder",
} as const satisfies Record<CreatableAssetKind, string>;

export function CreateAssetForm({
  kind,
  onCancel,
  onCreate,
  error,
  isSubmitting = false,
}: {
  kind: CreatableAssetKind;
  onCancel: () => void;
  onCreate: (request: CreationRequest<File>) => void | Promise<void>;
  error?: Error | null;
  isSubmitting?: boolean;
}) {
  const { t } = useTranslation(["generation", "common"]);
  const [validationError, setValidationError] = useState<string>();
  const [localSubmissionReady, setLocalSubmissionReady] = useState(false);
  const [initialDraft] = useState(() => createAssetCreationDraft<File>(kind));
  const form = useForm({
    defaultValues: { draft: initialDraft },
    onSubmit: async ({ value }) => {
      const result = assetCreationDraftSchema.safeParse(value.draft);
      if (!result.success) {
        setValidationError(
          [...new Set(result.error.issues.map((issue) => issue.message))].join(
            " ",
          ),
        );
        return;
      }

      setValidationError(undefined);
      const validatedDraft = result.data as AssetCreationDraft<File>;
      if (validatedDraft.kind === "uiset") {
        setLocalSubmissionReady(true);
        return;
      }
      await onCreate(toCreationRequest(validatedDraft));
    },
  });
  const draft = useStore(form.store, (state) => state.values.draft);
  const setDraft = (nextDraft: AssetCreationDraft<File>) => {
    setValidationError(undefined);
    setLocalSubmissionReady(false);
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
          {t("assetName")}
          <Input
            required
            placeholder={t(assetNamePlaceholderKeys[draft.kind])}
            value={draft.name}
            onChange={(event) =>
              setDraft({ ...draft, name: event.target.value })
            }
          />
        </label>
        <label className="grid gap-2 text-sm font-medium lg:col-span-2">
          {t("creativeBrief")}
          <Textarea
            required
            className="min-h-28 resize-none"
            placeholder={
              draft.kind === "audio"
                ? t("audioPromptPlaceholder")
                : draft.kind === "scenery"
                  ? t("sceneryPromptPlaceholder")
                  : t("promptPlaceholder")
            }
            value={draft.prompt}
            onChange={(event) =>
              setDraft({ ...draft, prompt: event.target.value })
            }
          />
        </label>
      </div>

      <AssetSpecificFields draft={draft} onChange={setDraft} />

      {validationError ? (
        <p className="text-sm text-destructive" role="alert">
          {validationError}
        </p>
      ) : null}

      {localSubmissionReady ? (
        <p className="text-sm text-muted-foreground" role="status">
          {t("uiSetLocalSubmissionReady")}
        </p>
      ) : null}

      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error.message || t("createError")}
        </p>
      ) : null}

      <div className="flex flex-col-reverse gap-2 border-t pt-5 sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          disabled={isSubmitting}
          onClick={onCancel}
        >
          {t("common:actions.cancel")}
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <LoaderCircle className="animate-spin" /> : null}
          {isSubmitting
            ? t("common:actions.creating")
            : t("createKind", { kind: t(`common:assetKinds.${kind}`) })}
        </Button>
      </div>
    </form>
  );
}

function AssetSpecificFields({
  draft,
  onChange,
}: {
  draft: AssetCreationDraft<File>;
  onChange: (draft: AssetCreationDraft<File>) => void;
}) {
  switch (draft.kind) {
    case "tileset":
      return <TilesetAssetFields draft={draft} onChange={onChange} />;
    case "scenery":
      return <SceneryAssetFields draft={draft} onChange={onChange} />;
    case "uiset":
      return <UISetAssetFields draft={draft} onChange={onChange} />;
    case "character":
    case "object":
      return <VisualAssetFields draft={draft} onChange={onChange} />;
    case "audio":
      return null;
  }
}
