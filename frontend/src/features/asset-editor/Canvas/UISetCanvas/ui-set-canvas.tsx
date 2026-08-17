import { PanelsTopLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { CSSProperties } from "react";

import { cn } from "@/lib/utils";
import type { UISetComponent } from "@/model";

export type UISetCanvasModel = {
  components: readonly UISetComponent[];
  selectedComponentId: string | null;
};

export type UISetCanvasEvent = {
  type: "component.selected";
  componentId: string;
};

export type UISetCanvasProps = {
  model: UISetCanvasModel;
  onEvent: (event: UISetCanvasEvent) => void;
};

export function UISetCanvas({ model, onEvent }: UISetCanvasProps) {
  const { t } = useTranslation("editor");
  const selectedComponent = model.components.find(
    (component) => component.id === model.selectedComponentId,
  );

  return (
    <main className="min-h-0 min-w-0 flex-1 overflow-hidden bg-transparent p-4 sm:p-6 lg:p-8">
      <section
        aria-label={t("uiSetCanvas")}
        className="flex h-full min-h-[24rem] items-center justify-center lg:min-h-[36rem]"
      >
        <div className="relative aspect-video w-full max-w-[72rem] overflow-hidden">
          {model.components.length === 0 ? <EmptyUISetCanvas /> : null}
          {model.components.map((component) => {
            const selected = component.id === model.selectedComponentId;

            return (
              <button
                key={component.id}
                type="button"
                aria-label={`Select ${component.label}`}
                aria-pressed={selected}
                onClick={() =>
                  onEvent({
                    type: "component.selected",
                    componentId: component.id,
                  })
                }
                className={cn(
                  "absolute overflow-hidden border text-left transition-[border-color,box-shadow] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#d99096]",
                  getUISetComponentClassName(component.kind, selected),
                )}
                style={getUISetComponentStyle(component)}
              >
                <UISetComponentContent component={component} />
              </button>
            );
          })}
        </div>
      </section>
      <p className="sr-only" aria-live="polite">
        {selectedComponent
          ? t("componentsSelected", { value: selectedComponent.label })
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
    ? "border-[#f8d477] ring-2 ring-[#f8d477]/75"
    : "border-transparent hover:border-[#f8d477]/75";

  switch (kind) {
    case "panel":
      return cn(
        "flex items-start rounded-[1.1rem] border-2 bg-[#f4dfaa] p-3 text-[#4b2b18] shadow-[0_14px_30px_rgb(6_18_28/0.35)] sm:p-5",
        selectionClassName,
      );
    case "label":
      return cn(
        "flex items-center rounded-md bg-[#1f343a]/90 px-2 text-[#fff4d2] shadow-[0_3px_0_rgb(43_22_10/0.35)] backdrop-blur-sm",
        selectionClassName,
      );
    case "button":
      return cn(
        "flex items-center justify-center rounded-lg border-2 border-[#f4ca65] bg-[#9b4d2d] text-center text-[#fff1c7] shadow-[0_5px_0_rgb(63_27_13/0.8),0_10px_18px_rgb(9_22_31/0.28)] transition-[background-color,transform,box-shadow] hover:-translate-y-0.5 hover:bg-[#b35b35] active:translate-y-0 active:shadow-[0_2px_0_rgb(63_27_13/0.8)]",
        selectionClassName,
      );
    case "input":
      return cn(
        "flex items-center rounded-md border-2 border-[#b98b52] bg-[#fff4d2] px-3 text-[#70401f] shadow-inner",
        selectionClassName,
      );
    case "badge":
      return cn(
        "flex items-center justify-center rounded-full border border-[#f4ca65] bg-[#d99096] px-3 text-xs font-bold text-[#4b2b18]",
        selectionClassName,
      );
    case "progress":
      return cn(
        "flex items-center rounded-full border border-[#b98b52] bg-[#fff4d2] p-1",
        selectionClassName,
      );
    case "toggle":
      return cn(
        "flex items-center rounded-full border-2 border-[#f4ca65] bg-[#9b4d2d] p-1 text-[#fff1c7]",
        selectionClassName,
      );
    case "icon":
      return cn(
        "grid place-items-center rounded-md border-2 border-[#f4ca65] bg-[#9b4d2d] text-[#fff1c7] shadow-[0_4px_0_rgb(63_27_13/0.8)]",
        selectionClassName,
      );
    case "slider":
      return cn(
        "flex items-center rounded-full border border-[#b98b52] bg-[#fff4d2] px-1.5",
        selectionClassName,
      );
  }
}

function UISetComponentContent({ component }: { component: UISetComponent }) {
  if (component.kind === "panel") {
    return (
      <span className="pointer-events-none flex w-full flex-col gap-3">
        <span className="flex items-center gap-1.5 border-b border-[#9d6938]/35 pb-2 text-[0.58rem] font-black uppercase tracking-[0.18em] text-[#70401f] sm:text-xs">
          <span className="size-1.5 rounded-full bg-[#b86b38]" />
          <span className="size-1.5 rounded-full bg-[#d49d42]" />
          <span className="size-1.5 rounded-full bg-[#e0bd67]" />
          <span className="ml-1 truncate">{component.label}</span>
        </span>
      </span>
    );
  }

  if (component.kind === "input") {
    return (
      <span className="pointer-events-none truncate text-xs font-semibold">
        {component.label}
      </span>
    );
  }
  if (component.kind === "progress") {
    return (
      <span className="pointer-events-none block h-full w-3/5 rounded-full bg-[#d99096]" />
    );
  }
  if (component.kind === "toggle") {
    return (
      <>
        <span className="pointer-events-none size-4 rounded-full bg-[#fff1c7]" />
        <span className="sr-only">{component.label}</span>
      </>
    );
  }
  if (component.kind === "icon") {
    return (
      <span className="pointer-events-none size-1/2 rotate-45 rounded-sm border-2 border-current" />
    );
  }
  if (component.kind === "slider") {
    return (
      <span className="pointer-events-none relative h-1 w-full rounded-full bg-[#d99096] before:absolute before:top-1/2 before:left-2/3 before:size-3 before:-translate-x-1/2 before:-translate-y-1/2 before:rounded-full before:border-2 before:border-[#9b4d2d] before:bg-[#fff1c7]" />
    );
  }
  if (component.kind === "badge") {
    return (
      <span className="pointer-events-none truncate">{component.label}</span>
    );
  }
  return (
    <span className="pointer-events-none block max-h-full max-w-full px-2 text-xs leading-tight font-black [overflow-wrap:anywhere] sm:px-3 sm:text-base">
      {component.label}
    </span>
  );
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
