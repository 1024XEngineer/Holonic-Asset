import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createAnimatedSpriteCanvasActions } from "./animated-sprite-canvas-events";
import { getAnimatedSpriteNodeLabel } from "./animated-sprite-node";
import { AnimatedSpriteCanvasLoading } from "./Loading/animated-sprite-canvas-loading";
import { AnimatedSpriteCanvasRuntime } from "./Runtime/AnimatedSpriteCanvasRuntime";
import {
  FRAME_SIZE,
  INITIAL_SCALE,
  MAX_SOURCE_PIXEL_SCREEN_SIZE,
  MIN_SCALE,
} from "./Runtime/AnimatedSpriteStage.constants";
import { AnimatedSpriteZoomControl } from "./animated-sprite-zoom-control";
import { getAnimatedSpriteMaxScale } from "./animated-sprite-scale";
import type { AnimatedSpriteCanvasProps } from "./AnimatedSpriteCanvas.interface";

export function AnimatedSpriteCanvas({
  model,
  onEvent,
}: AnimatedSpriteCanvasProps) {
  const { t } = useTranslation("editor");
  const hostRef = useRef<HTMLDivElement>(null);
  const runtimeRef = useRef<AnimatedSpriteCanvasRuntime>(null);
  const [loading, setLoading] = useState(true);
  const [zoom, setZoom] = useState(INITIAL_SCALE);
  const maxZoom = getAnimatedSpriteMaxScale(
    model.prototype,
    FRAME_SIZE,
    MAX_SOURCE_PIXEL_SCREEN_SIZE,
  );
  const runtimeProps = {
    model,
    actions: createAnimatedSpriteCanvasActions(onEvent),
    onZoomChange: setZoom,
  };
  const selectionLabel = model.selection.nodeIds
    .map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, model.animations))
    .join(", ");

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const runtime = new AnimatedSpriteCanvasRuntime(runtimeProps);
    runtimeRef.current = runtime;
    let disposed = false;
    void runtime
      .initialize(host)
      .then(() => {
        if (disposed) runtime.destroy();
        else setLoading(false);
      })
      .catch(() => {
        if (!disposed) setLoading(false);
      });
    return () => {
      disposed = true;
      runtimeRef.current = null;
      runtime.destroy();
    };
  }, []);

  useEffect(() => {
    runtimeRef.current?.syncProps(runtimeProps);
  }, [runtimeProps]);

  return (
    <main className="relative min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7]">
      <div ref={hostRef} className="size-full cursor-default" />
      {loading ? <AnimatedSpriteCanvasLoading /> : null}
      {!loading ? (
        <AnimatedSpriteZoomControl
          scale={zoom}
          minScale={MIN_SCALE}
          maxScale={maxZoom}
          onScaleChange={(scale) => runtimeRef.current?.setZoom(scale)}
        />
      ) : null}
      <p className="sr-only" aria-live="polite">
        {selectionLabel
          ? t("selectionSummary", { value: selectionLabel })
          : t("noCanvasItems")}
      </p>
    </main>
  );
}
