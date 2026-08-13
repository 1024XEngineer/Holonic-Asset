import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Input } from "@/components/ui/input";

type AnimatedSpriteZoomControlProps = {
  scale: number;
  minScale: number;
  maxScale: number;
  onScaleChange: (scale: number) => void;
};

export function AnimatedSpriteZoomControl({
  scale,
  minScale,
  maxScale,
  onScaleChange,
}: AnimatedSpriteZoomControlProps) {
  const { t } = useTranslation("editor");
  const percentage = scaleToPercentage(scale);
  const [value, setValue] = useState(String(percentage));
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    if (!editing) setValue(String(percentage));
  }, [editing, percentage]);

  const applyValue = () => {
    const parsed = Number(value);
    if (value.trim() === "" || !Number.isFinite(parsed)) {
      setValue(String(percentage));
      return;
    }

    const nextPercentage = clamp(
      Math.round(parsed),
      scaleToPercentage(minScale),
      scaleToPercentage(maxScale),
    );
    setValue(String(nextPercentage));
    onScaleChange(nextPercentage / 100);
  };

  return (
    <div className="absolute right-3 bottom-3 flex h-8 items-center rounded-lg border border-black/10 bg-background/95 px-2 shadow-sm backdrop-blur-sm">
      <Search
        className="mr-1.5 size-3.5 text-muted-foreground"
        aria-hidden="true"
      />
      <label htmlFor="animated-sprite-canvas-zoom" className="sr-only">
        {t("canvasZoom")}
      </label>
      <Input
        id="animated-sprite-canvas-zoom"
        type="number"
        inputMode="numeric"
        min={scaleToPercentage(minScale)}
        max={scaleToPercentage(maxScale)}
        step={1}
        value={value}
        aria-label={t("canvasZoom")}
        title={t("canvasZoom")}
        onFocus={() => setEditing(true)}
        onChange={(event) => setValue(event.target.value)}
        onBlur={() => {
          applyValue();
          setEditing(false);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") event.currentTarget.blur();
        }}
        className="h-6 w-12 appearance-none border-0 bg-transparent px-0 py-0 text-right text-xs font-medium tabular-nums shadow-none focus-visible:ring-0 [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
      />
      <span className="ml-0.5 text-xs text-muted-foreground" aria-hidden="true">
        %
      </span>
    </div>
  );
}

function scaleToPercentage(scale: number) {
  return Math.round(scale * 100);
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}
