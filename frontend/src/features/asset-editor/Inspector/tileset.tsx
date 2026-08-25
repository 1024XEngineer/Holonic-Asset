import { useTranslation } from "react-i18next";

import { EditPromptForm } from "./edit-prompt-form";
import type { TilesetInspectorProps } from "./inspector.types";
import { useEditPrompt } from "./use-inspector-edit";

export function TilesetInspectorContent({
  prompt,
  target,
  targetError,
  isSubmitting = false,
  onPromptChange,
  onSubmit,
  onClearSelection,
}: Omit<TilesetInspectorProps, "history" | "kind">) {
  const { t } = useTranslation("editor");
  const controller = useEditPrompt({
    prompt,
    onPromptChange,
    onSubmit,
    isSubmitting,
    target,
    canSubmitTarget: target !== null && targetError === null,
  });

  return (
    <div>
      {targetError ? (
        <p className="mb-3 rounded-lg border bg-muted/30 p-3 text-xs leading-5 text-muted-foreground">
          {targetError}
        </p>
      ) : null}
      <EditPromptForm
        controller={controller}
        prompt={prompt}
        isSubmitting={isSubmitting}
        onClearSelection={onClearSelection}
        clearSelectionTooltip={t("clearTilesetSelection")}
      />
    </div>
  );
}
