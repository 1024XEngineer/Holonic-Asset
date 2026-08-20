// @vitest-environment happy-dom

import { describe, expect, it, vi } from "vitest";

import type { AnimatedSpriteCanvasActions } from "../Runtime/AnimatedSpriteCanvas.types";
import { AnimatedSpriteStageInteraction } from "./AnimatedSpriteStageInteraction";

const animations = [
  { kind: "clip" as const, id: "idle", label: "Idle", frameCount: 4 },
];

function setup() {
  const canvas = document.createElement("canvas");
  const captured = new Set<number>();
  canvas.setPointerCapture = vi.fn((pointerId) => captured.add(pointerId));
  canvas.hasPointerCapture = vi.fn((pointerId) => captured.has(pointerId));
  canvas.releasePointerCapture = vi.fn((pointerId) =>
    captured.delete(pointerId),
  );
  canvas.getBoundingClientRect = vi.fn(
    () =>
      ({
        left: 0,
        top: 0,
        right: 800,
        bottom: 600,
        width: 800,
        height: 600,
        x: 0,
        y: 0,
        toJSON: vi.fn(),
      }) as DOMRect,
  );
  const actions: AnimatedSpriteCanvasActions = {
    onSelect: vi.fn(),
    onSelectFrame: vi.fn(),
    onSelectFrames: vi.fn(),
    onSelectNodes: vi.fn(),
    onClearSelection: vi.fn(),
    onNodePositionChange: vi.fn(),
    onReviewResolve: vi.fn(),
  };
  const interaction = new AnimatedSpriteStageInteraction(canvas, {
    viewport: { toWorld: (point: { x: number; y: number }) => point } as never,
    actions,
    getAnimations: () => animations,
    getPrototype: () => ({ columns: 1, rows: 1 }),
    getScene: () => ({
      positions: {
        prototype: { x: 20, y: 400 },
        idle: { x: 300, y: 30 },
      },
      expanded: new Set(["idle"]),
      playing: new Set(),
      previewFrames: new Map(),
      marquee: null,
    }),
    getReview: () => undefined,
    moveNode: vi.fn(),
    setMarquee: vi.fn(),
    getDragStep: () => 1,
    toggleExpanded: vi.fn(),
    togglePlaying: vi.fn(),
    render: vi.fn(),
  });
  return { actions, canvas, interaction };
}

function pointer(
  canvas: HTMLCanvasElement,
  type: "pointerdown" | "pointermove" | "pointerup",
  x: number,
  y: number,
  options: PointerEventInit = {},
) {
  canvas.dispatchEvent(
    new PointerEvent(type, {
      bubbles: true,
      button: 0,
      buttons: type === "pointerup" ? 0 : 1,
      pointerId: 1,
      clientX: x,
      clientY: y,
      ...options,
    }),
  );
}

describe("AnimatedSpriteStageInteraction", () => {
  it("drag-selects multiple frames when the drag starts on a frame", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 310, 80);
    pointer(canvas, "pointermove", 425, 100);
    pointer(canvas, "pointerup", 425, 100);

    expect(actions.onSelectFrame).not.toHaveBeenCalled();
    expect(actions.onSelectFrames).toHaveBeenCalledWith(
      [
        { nodeId: "idle", index: 0 },
        { nodeId: "idle", index: 1 },
      ],
      false,
    );
    interaction.destroy();
  });

  it("passes Ctrl/Command selection intent for frame and animation clicks", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 310, 80, { metaKey: true });
    pointer(canvas, "pointerup", 310, 80, { metaKey: true });
    pointer(canvas, "pointerdown", 310, 45, { ctrlKey: true });

    expect(actions.onSelectFrame).toHaveBeenCalledWith("idle", 0, true);
    expect(actions.onSelect).toHaveBeenCalledWith("idle", true);
    interaction.destroy();
  });
});
