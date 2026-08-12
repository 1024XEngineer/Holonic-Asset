import { Button } from "@/components/ui/button";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { UISetAssetCreationDraft } from "../types";
import { useTranslation } from "react-i18next";

export function UISetAssetFields({
  draft,
  onChange,
}: {
  draft: UISetAssetCreationDraft<File>;
  onChange: (draft: UISetAssetCreationDraft<File>) => void;
}) {
  const { t } = useTranslation("generation");
  const updateComponent = (
    index: number,
    patch: Partial<UISetAssetCreationDraft<File>["components"][number]>,
  ) =>
    onChange({
      ...draft,
      components: draft.components.map((component, componentIndex) =>
        componentIndex === index ? { ...component, ...patch } : component,
      ),
    });

  return (
    <>
      <div className="grid gap-3">
        <p className="text-sm font-medium">{t("layoutComponents")}</p>
        <div className="grid max-h-80 gap-3 overflow-y-auto pr-1">
          {draft.components.map((component, index) => (
            <div key={index} className="grid gap-2 rounded-lg border p-3">
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={component.isCustom}
                  onChange={(event) =>
                    updateComponent(index, { isCustom: event.target.checked })
                  }
                />
                {t("customComponent")}
              </label>
              <Input
                required
                placeholder={t("componentName")}
                value={component.name}
                onChange={(event) =>
                  updateComponent(index, { name: event.target.value })
                }
              />
              <Textarea
                required
                className="resize-none"
                placeholder={
                  component.isCustom
                    ? t("customComponentPlaceholder")
                    : t("componentPlaceholder")
                }
                value={component.description}
                onChange={(event) =>
                  updateComponent(index, { description: event.target.value })
                }
              />
            </div>
          ))}
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            onChange({
              ...draft,
              components: [
                ...draft.components,
                { name: "", description: "", isCustom: false },
              ],
            })
          }
        >
          {t("addComponent")}
        </Button>
      </div>
      <label className="grid gap-2 text-sm font-medium">
        {t("style")}
        <Textarea
          required
          className="min-h-20 resize-none"
          placeholder={t("uiStylePlaceholder")}
          value={draft.style}
          onChange={(event) =>
            onChange({ ...draft, style: event.target.value })
          }
        />
      </label>
      <ImageDropzone
        fileName={draft.reference?.name}
        onSelect={(reference) => onChange({ ...draft, reference })}
        onClear={() => onChange({ ...draft, reference: undefined })}
      />
    </>
  );
}
