import { useEffect, useRef, useState } from "react";

import { createCharacterCanvasActions } from "./character-canvas-events";
import { getCharacterNodeLabel } from "./character-node";
import { CharacterCanvasLoading } from "./Loading/CharacterCanvasLoading";
import { CharacterCanvasRuntime } from "./Runtime/CharacterCanvasRuntime";
import type { CharacterCanvasProps } from "./CharacterCanvas.interface";

export function CharacterCanvas({ model, onEvent }: CharacterCanvasProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const runtimeRef = useRef<CharacterCanvasRuntime>(null);
  const [loading, setLoading] = useState(true);
  const runtimeProps = {
    model,
    actions: createCharacterCanvasActions(onEvent),
  };

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const runtime = new CharacterCanvasRuntime(runtimeProps);
    runtimeRef.current = runtime;
    let disposed = false;
    let initialized = false;
    void runtime
      .initialize(host)
      .then(() => {
        initialized = true;
        if (disposed) {
          runtime.destroy();
          return;
        }
        setLoading(false);
      })
      .catch(() => {
        if (!disposed) setLoading(false);
      });
    return () => {
      disposed = true;
      runtimeRef.current = null;
      if (initialized) runtime.destroy();
    };
  }, []);

  useEffect(() => {
    runtimeRef.current?.syncProps(runtimeProps);
  }, [runtimeProps]);

  return (
    <main className="relative min-h-0 min-w-0 flex-1 overflow-hidden bg-[#eeece7]">
      <div
        ref={hostRef}
        className="size-full cursor-default data-[panning=true]:cursor-grabbing"
      />
      {loading ? <CharacterCanvasLoading /> : null}
      <p className="sr-only" aria-live="polite">
        {model.selection.nodeIds.length > 0
          ? `${model.selection.nodeIds.map((nodeId) => getCharacterNodeLabel(nodeId, model.animations)).join(", ")} selected`
          : "No canvas items selected"}
      </p>
    </main>
  );
}
