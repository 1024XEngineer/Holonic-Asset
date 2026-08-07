import { useEffect, useRef, useState } from "react";
import { createAnimatedSpriteCanvasActions } from "./animated-sprite-canvas-events";
import { getAnimatedSpriteNodeLabel } from "./animated-sprite-node";
import { AnimatedSpriteCanvasLoading } from "./Loading/animated-sprite-canvas-loading";
import { AnimatedSpriteCanvasRuntime } from "./Runtime/AnimatedSpriteCanvasRuntime";
import type { AnimatedSpriteCanvasProps } from "./AnimatedSpriteCanvas.interface";

export function AnimatedSpriteCanvas({
  model,
  onEvent,
}: AnimatedSpriteCanvasProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const runtimeRef = useRef<AnimatedSpriteCanvasRuntime>(null);
  const [loading, setLoading] = useState(true);
  const runtimeProps = {
    model,
    actions: createAnimatedSpriteCanvasActions(onEvent),
  };

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
      <p className="sr-only" aria-live="polite">
        {model.selection.nodeIds.length > 0
          ? `${model.selection.nodeIds.map((nodeId) => getAnimatedSpriteNodeLabel(nodeId, model.animations)).join(", ")} selected`
          : "No canvas items selected"}
      </p>
    </main>
  );
}
