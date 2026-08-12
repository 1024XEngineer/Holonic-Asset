import { PanelsTopLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { CSSProperties } from "react";

import { cn } from "@/lib/utils";
import type { UISetComponent } from "@/model";

export type UISetCanvasModel = {
  components: readonly UISetComponent[];
  selectedComponentIds: readonly string[];
};

export type UISetCanvasEvent = {
  type: "component.selection.toggled";
  componentId: string;
};

export type UISetCanvasProps = {
  model: UISetCanvasModel;
  onEvent: (event: UISetCanvasEvent) => void;
};

export function UISetCanvas({ model, onEvent }: UISetCanvasProps) {
  const { t } = useTranslation("editor");
  const selectedComponents = new Set(model.selectedComponentIds);
  const selectionLabel = model.components
    .filter((component) => selectedComponents.has(component.id))
    .map((component) => component.label)
    .join(", ");

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7] p-4 sm:p-6 lg:p-8">
      <section
        aria-label={t("uiSetCanvas")}
        className="flex h-full min-h-[24rem] items-center justify-center lg:min-h-[36rem]"
      >
        <div className="relative aspect-video w-full max-w-[72rem] overflow-hidden border border-[#51493f]/25 bg-[#1f343a] shadow-[0_18px_50px_rgb(45_41_35/0.16)]">
          <div
            aria-hidden="true"
            className="absolute inset-0 opacity-20 [background-image:linear-gradient(90deg,transparent_49.75%,#d9f2ec_50%,transparent_50.25%),linear-gradient(transparent_49.75%,#d9f2ec_50%,transparent_50.25%)] [background-size:10%_10%]"
          />
          {model.components.length === 0 ? <EmptyUISetCanvas /> : null}
          {model.components.map((component) => {
            const selected = selectedComponents.has(component.id);

            return (
              <button
                key={component.id}
                type="button"
                aria-label={`Toggle selection for ${component.label}`}
                aria-pressed={selected}
                onClick={() =>
                  onEvent({
                    type: "component.selection.toggled",
                    componentId: component.id,
                  })
                }
                className={cn(
                  "absolute overflow-hidden border text-left transition-[border-color,box-shadow] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#d99096]",
                  getUISetComponentClassName(component.kind, selected),
                )}
                style={getUISetComponentStyle(component)}
              >
                <span className="pointer-events-none block max-h-full max-w-full px-2 text-xs leading-tight font-semibold [overflow-wrap:anywhere] sm:px-3">
                  {component.label}
                </span>
              </button>
            );
          })}
        </div>
      </section>
      <p className="sr-only" aria-live="polite">
        {selectionLabel
          ? t("componentsSelected", { value: selectionLabel })
          : t("noComponentsSelected")}
      </p>
    </main>
  );
}

function getUISetComponentStyle(component: UISetComponent): CSSProperties {
  const left = clampPercent(component.bounds.x);
  const top = clampPercent(component.bounds.y);
  const width = Math.min(clampPercent(component.bounds.width), 100 - left);
  const height = Math.min(clampPercent(component.bounds.height), 100 - top);

  return {
    left: `${left}%`,
    top: `${top}%`,
    width: `${width}%`,
    height: `${height}%`,
    zIndex: component.kind === "panel" ? 10 : 20,
  };
}

function getUISetComponentClassName(
  kind: UISetComponent["kind"],
  selected: boolean,
) {
  const selectionClassName = selected
    ? "border-[#d99096] ring-2 ring-[#d99096]/65"
    : "border-transparent hover:border-[#d99096]/75";

  switch (kind) {
    case "panel":
      return cn(
        "flex items-start rounded-md bg-[#f7f5f0] pt-3 text-[#2d2923] shadow-lg",
        selectionClassName,
      );
    case "label":
      return cn(
        "flex items-center bg-transparent text-[#2d2923]",
        selectionClassName,
      );
    case "button":
      return cn(
        "flex items-center justify-center rounded-sm bg-[#b86b70] text-center text-white shadow-sm",
        selectionClassName,
      );
  }
}

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

function EmptyUISetCanvas() {
  const { t } = useTranslation("editor");
  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-[#a8c5c6]">
      <PanelsTopLeft className="size-5" aria-hidden="true" />
      <p className="text-xs font-medium">{t("noUISetComponents")}</p>
    </div>
  );
}
