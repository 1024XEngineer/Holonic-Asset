import { ChevronDown, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ImageDropzone } from "@/components/ui/custom/image-dropzone";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { createUISetComponent } from "../lib";
import type { UISetAssetCreationDraft } from "../types";
import {
  uiSetCanvasHeightOptions,
  uiSetCanvasWidthOptions,
} from "./ui-set-canvas";
import { useTranslation } from "react-i18next";

export function UISetAssetFields({
  draft,
  onChange,
}: {
  draft: UISetAssetCreationDraft<File>;
  onChange: (draft: UISetAssetCreationDraft<File>) => void;
}) {
  const { t } = useTranslation("generation");
  const [expandedComponents, setExpandedComponents] = useState(
    () => new Set(draft.components.map((component) => component.id)),
  );
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
        <div>
          <p className="text-sm font-medium">{t("canvasSize")}</p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("uiCanvasSizeDescription")}
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="grid gap-2 text-sm font-medium">
            {t("canvasWidth")}
            <CanvasDimensionSelect
              value={draft.dimensions.width}
              label={t("canvasWidth")}
              options={uiSetCanvasWidthOptions}
              onChange={(width) =>
                onChange({
                  ...draft,
                  dimensions: { ...draft.dimensions, width },
                })
              }
            />
          </label>
          <label className="grid gap-2 text-sm font-medium">
            {t("canvasHeight")}
            <CanvasDimensionSelect
              value={draft.dimensions.height}
              label={t("canvasHeight")}
              options={uiSetCanvasHeightOptions}
              onChange={(height) =>
                onChange({
                  ...draft,
                  dimensions: { ...draft.dimensions, height },
                })
              }
            />
          </label>
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
        <div className="mt-3 flex items-end justify-between gap-3">
          <div>
            <p className="text-sm font-medium">{t("layoutComponents")}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("componentOrderDescription")}
            </p>
          </div>
          <span className="text-xs text-muted-foreground">
            {draft.components.length}
          </span>
        </div>
        <div className="grid gap-4">
          {draft.components.map((component, index) => {
            const expanded = expandedComponents.has(component.id);

            return (
              <section
                key={component.id}
                className="grid gap-3 rounded-lg border p-4"
                aria-label={t("componentItem", { number: index + 1 })}
              >
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-semibold">
                    {component.name ||
                      t("componentItem", { number: index + 1 })}
                  </h3>
                  <div className="flex items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      disabled={draft.components.length === 1}
                      title={t("removeComponent")}
                      aria-label={t("removeComponent")}
                      onClick={() => {
                        setExpandedComponents(
                          (current) =>
                            new Set(
                              [...current].filter(
                                (componentId) => componentId !== component.id,
                              ),
                            ),
                        );
                        onChange({
                          ...draft,
                          components: draft.components.filter(
                            (_, componentIndex) => componentIndex !== index,
                          ),
                        });
                      }}
                    >
                      <Trash2 />
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label={t(
                        expanded ? "collapseComponent" : "expandComponent",
                        { number: index + 1 },
                      )}
                      aria-expanded={expanded}
                      onClick={() =>
                        setExpandedComponents((current) => {
                          const next = new Set(current);
                          if (next.has(component.id)) next.delete(component.id);
                          else next.add(component.id);
                          return next;
                        })
                      }
                    >
                      <ChevronDown
                        className={`transition-transform ${expanded ? "" : "-rotate-90"}`}
                      />
                    </Button>
                  </div>
                </div>
                {expanded ? (
                  <>
                    <label className="grid gap-2 text-sm font-medium">
                      {t("componentName")}
                      <Input
                        required
                        placeholder={t("componentNamePlaceholder")}
                        value={component.name}
                        onChange={(event) =>
                          updateComponent(index, { name: event.target.value })
                        }
                      />
                    </label>
                    <label className="grid gap-2 text-sm font-medium">
                      {t("componentDescription")}
                      <Textarea
                        required
                        className="min-h-20 resize-none"
                        placeholder={t("componentDescriptionPlaceholder")}
                        value={component.description}
                        onChange={(event) =>
                          updateComponent(index, {
                            description: event.target.value,
                          })
                        }
                      />
                    </label>
                  </>
                ) : null}
              </section>
            );
          })}
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            const component = createUISetComponent();
            setExpandedComponents((current) =>
              new Set(current).add(component.id),
            );
            onChange({
              ...draft,
              components: [...draft.components, component],
            });
          }}
        >
          <Plus />
          {t("addComponent")}
        </Button>
      </div>
      <ImageDropzone
        label={t("uiSetCreatingReferenceImage")}
        value={draft.creatingReference}
        onChange={(creatingReference) =>
          onChange({ ...draft, creatingReference })
        }
      />
    </>
  );
}

function CanvasDimensionSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: number;
  options: readonly number[];
  onChange: (value: number) => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="outline"
            className="h-9 w-full justify-between px-3 font-normal"
            aria-label={label}
          />
        }
      >
        {value} px
        <ChevronDown className="size-4 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-(--anchor-width)">
        <DropdownMenuRadioGroup
          value={String(value)}
          onValueChange={(nextValue) => {
            onChange(Number(nextValue));
            setOpen(false);
          }}
        >
          {options.map((option) => (
            <DropdownMenuRadioItem key={option} value={String(option)}>
              {option} px
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
