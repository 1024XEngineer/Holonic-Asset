import { Layers3 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { cn } from "@/lib/utils";
import type { SceneryLayer } from "@/model";

import type { SceneryCanvasProps } from "./SceneryCanvas.interface";

const layerBlendClasses: Record<SceneryLayer["blendMode"], string> = {
  normal: "",
  multiply: "mix-blend-multiply",
};

export function SceneryCanvas({ model, onEvent }: SceneryCanvasProps) {
  const { t } = useTranslation("editor");
  const { layers, selectedLayerIds, visibleLayerIds } = model;
  const selectedLayers = new Set(selectedLayerIds);
  const visibleLayers = new Set(visibleLayerIds);
  const selectionLabel = layers
    .filter((layer) => selectedLayers.has(layer.id))
    .map((layer) => layer.label)
    .join(", ");

  return (
    <main
      aria-label="Scenery canvas"
      className="relative flex min-h-[24rem] min-w-0 flex-1 items-center justify-center overflow-hidden bg-[#eeece7] p-4 sm:p-6 lg:h-full lg:p-8"
    >
      <div className="relative aspect-video w-full max-w-5xl overflow-hidden rounded-md border border-black/10 bg-[#c8e8ed] shadow-[0_18px_50px_rgb(45_41_35/0.14)]">
        {layers.length === 0 ? <EmptySceneryCanvas /> : null}
        {layers.map((layer) => {
          const selected = selectedLayers.has(layer.id);
          const visible = visibleLayers.has(layer.id);

          return (
            <button
              key={layer.id}
              type="button"
              aria-label={`Toggle selection for ${layer.label}`}
              aria-pressed={selected}
              aria-hidden={!visible}
              tabIndex={visible ? 0 : -1}
              onClick={() =>
                onEvent({
                  type: "layer.selection.toggled",
                  layerId: layer.id,
                })
              }
              className={cn(
                "absolute inset-0 transition-[filter,opacity] duration-200 focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-offset-[-3px] focus-visible:outline-[#b86b70]",
                layerBlendClasses[layer.blendMode],
                !visible && "invisible opacity-0",
                selected &&
                  "brightness-110 saturate-125 drop-shadow-[0_0_8px_rgb(184_107_112/0.9)]",
              )}
            >
              <img
                src={layer.imageUrl}
                alt=""
                draggable={false}
                className="size-full object-cover"
                decoding="async"
              />
            </button>
          );
        })}
      </div>
      <p className="sr-only" aria-live="polite">
        {selectionLabel
          ? t("selectionSummary", { value: selectionLabel })
          : t("noLayersSelected")}
      </p>
    </main>
  );
}

function EmptySceneryCanvas() {
  const { t } = useTranslation("editor");
  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-[#47656a]">
      <Layers3 className="size-5" aria-hidden="true" />
      <p className="text-xs font-medium">{t("noSceneryLayers")}</p>
    </div>
  );
}
