import { Eye, EyeOff, Image, Layers3, MapPin, RotateCw } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { SceneryCanvasDimensions, SceneryLayer } from "@/model";

export function SceneryInspectorContent({
  layer,
  dimensions,
  visible,
  onToggleVisibility,
}: {
  layer: SceneryLayer | null;
  dimensions?: SceneryCanvasDimensions;
  visible: boolean;
  onToggleVisibility: () => void;
}) {
  const { t } = useTranslation("editor");

  if (!layer) {
    return (
      <div className="grid min-h-64 place-items-center rounded-xl border border-dashed p-6 text-center text-xs text-muted-foreground">
        {t("selectSceneryLayer")}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-xl border bg-muted/20">
        <div className="aspect-video bg-[#c8e8ed] p-2">
          <img
            src={layer.imageUrl}
            alt=""
            className="size-full object-contain"
          />
        </div>
        <div className="border-t px-3 py-3">
          <p className="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
            {t("selectedLayer")}
          </p>
          <h2 className="mt-1 truncate text-sm font-semibold">{layer.label}</h2>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            {layer.detail}
          </p>
        </div>
      </div>

      <button
        type="button"
        aria-pressed={visible}
        onClick={onToggleVisibility}
        className="flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-xs font-medium hover:bg-muted"
      >
        <span className="flex items-center gap-2">
          {visible ? <Eye className="size-4" /> : <EyeOff className="size-4" />}
          {visible ? t("visible") : t("hidden")}
        </span>
        <span className="text-muted-foreground">{t("toggle")}</span>
      </button>

      <div className="grid gap-2 rounded-lg border p-3 text-xs">
        <InspectorValue
          icon={<MapPin className="size-3.5" />}
          label={t("position")}
          value={formatPosition(layer.position)}
        />
        <InspectorValue
          icon={<RotateCw className="size-3.5" />}
          label={t("transform")}
          value={formatTransform(layer)}
        />
        <InspectorValue
          icon={<Image className="size-3.5" />}
          label={t("opacity")}
          value={`${Math.round((layer.opacity ?? 1) * 100)}%`}
        />
        <InspectorValue
          icon={<Layers3 className="size-3.5" />}
          label={t("layerOrder")}
          value={String(layer.zIndex ?? 0)}
        />
      </div>

      {dimensions ? (
        <p className="text-[11px] text-muted-foreground">
          {t("canvasDimensions", dimensions)}
        </p>
      ) : null}
    </div>
  );
}

function InspectorValue({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground">{icon}</span>
      <span className="text-muted-foreground">{label}</span>
      <span className="ml-auto font-mono text-[11px]">{value}</span>
    </div>
  );
}

function formatPosition(position?: { x: number; y: number }) {
  if (!position) return "0, 0";
  return `${Math.round(position.x)}, ${Math.round(position.y)}`;
}

function formatTransform(layer: SceneryLayer) {
  const scale = layer.transform?.scale ?? { x: 1, y: 1 };
  const rotation = layer.transform?.rotation ?? 0;
  return `${scale.x.toFixed(2)} x ${scale.y.toFixed(2)} / ${Math.round(rotation)}°`;
}
