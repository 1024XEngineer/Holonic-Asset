import { Eye, EyeOff, Layers3 } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { SceneryLayer } from "@/model";

import { AssetTree } from "./asset-tree";

export function SceneryAssetTree({
  layers,
  selectedLayerId,
  visibleLayerIds,
  onSelect,
  onToggleVisibility,
}: {
  layers: readonly SceneryLayer[];
  selectedLayerId: string | null;
  visibleLayerIds: readonly string[];
  onSelect: (layerId: string) => void;
  onToggleVisibility: (layerId: string) => void;
}) {
  const { t } = useTranslation("editor");
  const visible = new Set(visibleLayerIds);

  return (
    <AssetTree
      title={t("sceneryLayers")}
      description={t("sceneryLayersDescription")}
      count={layers.length}
      emptyMessage={layers.length === 0 ? t("noSceneryLayers") : undefined}
      contentClassName="space-y-1"
    >
      {[...layers].reverse().map((layer, index) => {
        const isVisible = visible.has(layer.id);
        const selected = selectedLayerId === layer.id;
        return (
          <div
            key={layer.id}
            className={`flex items-center gap-1 rounded-lg border transition-colors ${selected ? "border-primary/30 bg-primary/10" : "border-transparent hover:bg-muted"}`}
          >
            <button
              type="button"
              aria-current={selected ? "true" : undefined}
              onClick={() => onSelect(layer.id)}
              className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
            >
              <Layers3 className="size-4 shrink-0 text-primary" />
              <span className="min-w-0 flex-1 truncate text-xs font-medium">
                {layer.label}
              </span>
              {index === layers.length - 1 ? (
                <span className="text-[10px] text-muted-foreground">
                  {t("backdrop")}
                </span>
              ) : null}
            </button>
            <button
              type="button"
              aria-label={
                isVisible
                  ? t("hideLayer", { value: layer.label })
                  : t("showLayer", { value: layer.label })
              }
              aria-pressed={isVisible}
              title={isVisible ? t("hide") : t("show")}
              onClick={() => onToggleVisibility(layer.id)}
              className="mr-1 grid size-7 place-items-center rounded-md text-muted-foreground hover:bg-background hover:text-foreground"
            >
              {isVisible ? (
                <Eye className="size-3.5" />
              ) : (
                <EyeOff className="size-3.5" />
              )}
            </button>
          </div>
        );
      })}
    </AssetTree>
  );
}
