// @vitest-environment happy-dom

import { describe, expect, it, vi } from "vitest";

import type { AnimatedSpriteCanvasReview } from "../AnimatedSpriteCanvas.interface";
import type { AnimatedSpriteCanvasActions } from "../Runtime/AnimatedSpriteCanvas.types";
import { AnimatedSpriteStageInteraction } from "./AnimatedSpriteStageInteraction";

const animations = [
  { kind: "clip" as const, id: "idle", label: "Idle", frameCount: 4 },
];

function setup({
  expanded = true,
  review,
}: {
  expanded?: boolean;
  review?: AnimatedSpriteCanvasReview;
} = {}) {
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
  const onReviewResolve = vi.fn();
  const actions: AnimatedSpriteCanvasActions = {
    onSelect: vi.fn(),
    onSelectFrame: vi.fn(),
    onSelectFrames: vi.fn(),
    onSelectNodes: vi.fn(),
    onClearSelection: vi.fn(),
    onNodePositionChange: vi.fn(),
    onReviewResolve,
  };
  const positions: Record<string, { x: number; y: number }> = {
    prototype: { x: 20, y: 400 },
    idle: { x: 300, y: 30 },
  };
  const moveNode = vi.fn((node: string, position: { x: number; y: number }) => {
    positions[node] = position;
  });
  const setMarquee = vi.fn();
  const toggleExpanded = vi.fn();
  const togglePlaying = vi.fn();
  const render = vi.fn();
  const interaction = new AnimatedSpriteStageInteraction(canvas, {
    viewport: { toWorld: (point: { x: number; y: number }) => point } as never,
    actions,
    getAnimations: () => animations,
    getPrototype: () => ({ columns: 1, rows: 1 }),
    getScene: () => ({
      positions,
      expanded: expanded ? new Set(["idle"]) : new Set(),
      playing: new Set(),
      previewFrames: new Map(),
      marquee: null,
    }),
    getReview: () => review,
    moveNode,
    setMarquee,
    getDragStep: () => 1,
    toggleExpanded,
    togglePlaying,
    render,
  });
  return {
    actions,
    canvas,
    interaction,
    moveNode,
    onReviewResolve,
    render,
    setMarquee,
    toggleExpanded,
    togglePlaying,
  };
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

  it("routes play, expand, and review controls without starting a drag", () => {
    const review: AnimatedSpriteCanvasReview = {
      kind: "new-animation",
      nodeId: "idle",
      isResolving: false,
    };
    const {
      actions,
      canvas,
      interaction,
      onReviewResolve,
      toggleExpanded,
      togglePlaying,
    } = setup({ expanded: false, review });

    pointer(canvas, "pointerdown", 340, 300);
    pointer(canvas, "pointerdown", 420, 300);
    pointer(canvas, "pointerdown", 340, 350);
    pointer(canvas, "pointerdown", 420, 350);

    expect(actions.onSelect).toHaveBeenNthCalledWith(1, "idle");
    expect(actions.onSelect).toHaveBeenNthCalledWith(2, "idle");
    expect(togglePlaying).toHaveBeenCalledWith("idle");
    expect(toggleExpanded).toHaveBeenCalledWith("idle");
    expect(onReviewResolve.mock.calls).toEqual([[true], [false]]);
    interaction.destroy();
  });

  it("clears a plain empty click but preserves selection with a modifier", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 780, 580);
    pointer(canvas, "pointerup", 780, 580);
    pointer(canvas, "pointerdown", 780, 580, { ctrlKey: true });
    pointer(canvas, "pointerup", 780, 580, { ctrlKey: true });

    expect(actions.onClearSelection).toHaveBeenCalledOnce();
    interaction.destroy();
  });

  it("does not change selection when the frame-grid gap is clicked", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 410, 100);
    pointer(canvas, "pointerup", 410, 100);

    expect(actions.onSelect).not.toHaveBeenCalled();
    expect(actions.onSelectFrame).not.toHaveBeenCalled();
    expect(actions.onClearSelection).not.toHaveBeenCalled();
    interaction.destroy();
  });

  it("selects intersecting animations when a marquee misses their frames", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 280, 20);
    pointer(canvas, "pointermove", 530, 50);
    pointer(canvas, "pointerup", 530, 50);

    expect(actions.onSelectNodes).toHaveBeenCalledWith(["idle"], false);
    interaction.destroy();
  });

  it("clears selection when a completed marquee misses every target", () => {
    const { actions, canvas, interaction } = setup();

    pointer(canvas, "pointerdown", 760, 500);
    pointer(canvas, "pointermove", 790, 540);
    pointer(canvas, "pointerup", 790, 540);

    expect(actions.onClearSelection).toHaveBeenCalledOnce();
    interaction.destroy();
  });

  it("moves and commits an animation node position", () => {
    const { actions, canvas, interaction, moveNode } = setup();

    pointer(canvas, "pointerdown", 310, 45);
    pointer(canvas, "pointermove", 320, 55);
    pointer(canvas, "pointerup", 320, 55);

    expect(moveNode).toHaveBeenCalledWith("idle", { x: 310, y: 40 });
    expect(actions.onNodePositionChange).toHaveBeenCalledWith("idle", {
      x: 310,
      y: 40,
    });
    interaction.destroy();
  });

  it("ignores non-primary clicks and prevents the context menu", () => {
    const { actions, canvas, interaction } = setup();
    const secondaryClick = new PointerEvent("pointerdown", {
      bubbles: true,
      button: 2,
      pointerId: 1,
      clientX: 310,
      clientY: 45,
    });
    const contextMenu = new MouseEvent("contextmenu", { cancelable: true });

    canvas.dispatchEvent(secondaryClick);
    canvas.dispatchEvent(contextMenu);

    expect(actions.onSelect).not.toHaveBeenCalled();
    expect(contextMenu.defaultPrevented).toBe(true);
    interaction.destroy();
  });
});
