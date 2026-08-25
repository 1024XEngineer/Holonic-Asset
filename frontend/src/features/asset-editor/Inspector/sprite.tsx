import { EditPromptForm } from "./edit-prompt-form";
import type { SpriteInspectorContentProps } from "./inspector.types";
import { useInspectorEdit } from "./use-inspector-edit";

export function SpriteInspectorContent(props: SpriteInspectorContentProps) {
  const controller = useInspectorEdit(props);
  return (
    <EditPromptForm
      controller={controller}
      prompt={props.prompt}
      isSubmitting={props.isSubmitting ?? false}
      onClearSelection={props.onClearSelection}
    />
  );
}
