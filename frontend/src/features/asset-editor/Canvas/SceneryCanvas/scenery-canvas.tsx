import type { EditorSceneryLayer } from "@/model";

import type { SceneryCanvasProps } from "./SceneryCanvas.interface";

const layerBlendClasses: Record<EditorSceneryLayer["blendMode"], string> = {
  normal: "",
  multiply: "mix-blend-multiply",
};

function layerClass(
  selectedLayerIds: string[],
  visibleLayerIds: string[],
  layerId: string,
) {
  const base = "absolute inset-0 transition-[filter,opacity] duration-200";

  if (!visibleLayerIds.includes(layerId)) return `${base} invisible opacity-0`;
  if (!selectedLayerIds.includes(layerId)) return base;

  return `${base} brightness-110 saturate-125 drop-shadow-[0_0_8px_rgba(239,176,180,0.95)]`;
}

export function SceneryCanvas({ model, onEvent }: SceneryCanvasProps) {
  const { layers, selectedLayerIds, visibleLayerIds } = model;

  return (
    <main className="relative flex min-h-[28rem] min-w-0 flex-1 items-center justify-center overflow-hidden bg-[#eceae4] p-5 lg:h-full lg:p-10">
      <div className="relative aspect-video w-full max-w-5xl overflow-hidden rounded-2xl border border-black/10 bg-[#c8e8ed] shadow-sm">
        {layers.map((layer) => (
          <button
            key={layer.id}
            type="button"
            aria-label={`Toggle selection for ${layer.label}`}
            aria-pressed={selectedLayerIds.includes(layer.id)}
            onClick={() =>
              onEvent({ type: "layer.selection.toggled", layerId: layer.id })
            }
            className={`${layerClass(selectedLayerIds, visibleLayerIds, layer.id)} ${layerBlendClasses[layer.blendMode]}`}
          >
            <img
              src={layer.imageUrl}
              alt=""
              className="size-full object-cover"
              decoding="async"
              loading="eager"
            />
          </button>
        ))}
        {selectedLayerIds.length ? (
          <div className="pointer-events-none absolute left-4 top-4 z-20 rounded-full border border-white/30 bg-black/35 px-3 py-1 text-[11px] font-medium text-white backdrop-blur-sm">
            {selectedLayerIds
              .map(
                (layerId) =>
                  layers.find((layer) => layer.id === layerId)?.label,
              )
              .filter(Boolean)
              .join(", ")}{" "}
            selected
          </div>
        ) : null}
      </div>
    </main>
  );
}
