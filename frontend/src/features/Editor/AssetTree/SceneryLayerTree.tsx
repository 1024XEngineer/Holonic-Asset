import { Eye, EyeOff, Layers3 } from "lucide-react";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { EditorSceneryLayer } from "@/domain/asset";

export function SceneryLayerTree({
  layers,
  selectedLayers,
  visibleLayers,
  onToggleLayer,
  onToggleVisibility,
}: {
  layers: EditorSceneryLayer[];
  selectedLayers: string[];
  visibleLayers: string[];
  onToggleLayer: (layer: string) => void;
  onToggleVisibility: (layer: string) => void;
}) {
  return (
    <aside className="flex w-full shrink-0 flex-col border-b border-black/10 bg-white lg:h-full lg:w-[16.5rem] lg:border-r lg:border-b-0">
      <ScrollArea className="max-h-[16rem] flex-1 lg:max-h-none">
        <div className="p-3">
          <div className="flex items-center gap-2 px-2 pb-3 text-[10px] font-semibold uppercase tracking-[0.16em] text-[#81786d]">
            <Layers3 className="size-3.5" />
            Layers
          </div>
          <div className="space-y-1.5">
            {layers.map((layer) => {
              const selected = selectedLayers.includes(layer.id);
              const visible = visibleLayers.includes(layer.id);
              return (
                <div
                  key={layer.id}
                  className={`flex w-full items-center rounded-xl border transition-colors ${selected ? "border-[#cf858a] bg-[#cf858a]/10" : "border-transparent hover:bg-black/[.04]"}`}
                >
                  <button
                    type="button"
                    aria-pressed={selected}
                    onClick={() => onToggleLayer(layer.id)}
                    className="flex min-w-0 flex-1 items-center gap-3 p-2 text-left"
                  >
                    <span className="relative h-11 w-14 shrink-0 overflow-hidden rounded-lg border border-black/10 bg-[#c8e8ed]">
                      <img
                        src={layer.imageUrl}
                        alt=""
                        className="size-full object-cover"
                      />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs font-semibold text-[#3e3831]">
                        {layer.label}
                      </span>
                      <span className="block truncate text-[10px] text-[#81786d]">
                        {layer.detail}
                      </span>
                    </span>
                  </button>
                  <button
                    type="button"
                    aria-label={`${visible ? "Hide" : "Show"} ${layer.label}`}
                    aria-pressed={visible}
                    onClick={() => onToggleVisibility(layer.id)}
                    className="mr-2 rounded-md p-1.5 text-[#81786d] hover:bg-black/[.06] hover:text-[#51493f]"
                  >
                    {visible ? (
                      <Eye className="size-4" />
                    ) : (
                      <EyeOff className="size-4 opacity-55" />
                    )}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </ScrollArea>
    </aside>
  );
}
