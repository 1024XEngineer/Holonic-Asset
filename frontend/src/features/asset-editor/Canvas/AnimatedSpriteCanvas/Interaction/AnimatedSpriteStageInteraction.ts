import type { NodeId } from "../animated-sprite-node";
import { intersectsRect, normalizeRect } from "@/lib/rect";
import { snapToStep } from "@/lib/snap-to-step";
import { getAnimatedSpriteFrameCount } from "../animated-sprite-frame-count";
import {
  getCanvasNodes,
  type CanvasPosition,
} from "../AnimatedSpriteCanvas.constants";
import {
  getFrameBounds,
  getNodeBounds,
  hitTestAnimatedSpriteScene,
} from "./AnimatedSpriteStageGeometry";
import type { AnimatedSpriteStageContext } from "../Runtime/AnimatedSpriteCanvas.types";

type DragState =
  | {
      kind: "node";
      pointerId: number;
      start: CanvasPosition;
      node: NodeId;
      position: CanvasPosition;
    }
  | {
      kind: "marquee";
      pointerId: number;
      start: CanvasPosition;
      end: CanvasPosition;
    }
  | {
      kind: "frame-marquee";
      pointerId: number;
      start: CanvasPosition;
      end: CanvasPosition;
      node: NodeId;
    };

export class AnimatedSpriteStageInteraction {
  private drag: DragState | null = null;
  private readonly canvas: HTMLCanvasElement;
  private readonly context: AnimatedSpriteStageContext;

  constructor(canvas: HTMLCanvasElement, context: AnimatedSpriteStageContext) {
    this.canvas = canvas;
    this.context = context;
    canvas.addEventListener("pointerdown", this.onPointerDown);
    canvas.addEventListener("pointermove", this.onPointerMove);
    canvas.addEventListener("pointerup", this.finishPointer);
    canvas.addEventListener("pointercancel", this.finishPointer);
    canvas.addEventListener("contextmenu", this.onContextMenu);
  }

  destroy() {
    this.canvas.removeEventListener("pointerdown", this.onPointerDown);
    this.canvas.removeEventListener("pointermove", this.onPointerMove);
    this.canvas.removeEventListener("pointerup", this.finishPointer);
    this.canvas.removeEventListener("pointercancel", this.finishPointer);
    this.canvas.removeEventListener("contextmenu", this.onContextMenu);
  }

  private onPointerDown = (event: PointerEvent) => {
    this.canvas.focus();
    if (event.button !== 0) return;
    const point = this.worldPoint(this.screenPoint(event));
    const hit = this.hitTest(point);
    if (hit?.kind === "play") return this.togglePlaying(hit.node);
    if (hit?.kind === "expand") return this.toggleExpanded(hit.node);
    if (hit?.kind === "review-apply")
      return this.context.actions.onReviewResolve(true);
    if (hit?.kind === "review-deny")
      return this.context.actions.onReviewResolve(false);
    if (hit?.kind === "frame")
      return this.context.actions.onSelectFrame(hit.node, hit.index);
    this.capture(event);
    if (hit?.kind === "frame-grid") {
      this.drag = {
        kind: "frame-marquee",
        pointerId: event.pointerId,
        start: point,
        end: point,
        node: hit.node,
      };
    } else if (hit?.kind === "node") {
      this.context.actions.onSelect(hit.node);
      this.drag = {
        kind: "node",
        pointerId: event.pointerId,
        start: point,
        node: hit.node,
        position: { ...this.context.getScene().positions[hit.node] },
      };
    } else {
      this.drag = {
        kind: "marquee",
        pointerId: event.pointerId,
        start: point,
        end: point,
      };
    }
    this.syncMarquee();
  };

  private onPointerMove = (event: PointerEvent) => {
    if (!this.drag || this.drag.pointerId !== event.pointerId) return;
    const point = this.worldPoint(this.screenPoint(event));
    if (this.drag.kind === "node") this.moveNode(this.drag, point);
    else this.drag.end = point;
    this.syncMarquee();
  };

  private finishPointer = (event: PointerEvent) => {
    if (!this.drag || this.drag.pointerId !== event.pointerId) return;
    const completed = this.drag;
    if (completed.kind === "marquee")
      this.completeNodeSelection(completed.start, completed.end);
    if (completed.kind === "frame-marquee")
      this.completeFrameSelection(
        completed.node,
        completed.start,
        completed.end,
      );
    if (completed.kind === "node")
      this.context.actions.onNodePositionChange(completed.node, {
        ...this.context.getScene().positions[completed.node],
      });
    this.drag = null;
    if (this.canvas.hasPointerCapture(event.pointerId))
      this.canvas.releasePointerCapture(event.pointerId);
    this.syncMarquee();
  };

  private onContextMenu = (event: MouseEvent) => event.preventDefault();

  private hitTest(point: CanvasPosition) {
    const model = this.context.getScene();
    return hitTestAnimatedSpriteScene(
      model,
      point,
      this.context.getPrototype(),
      this.context.getAnimations(),
      this.context.getReview(),
    );
  }

  private togglePlaying(node: NodeId) {
    this.context.actions.onSelect(node);
    this.context.togglePlaying(node);
    this.context.render();
  }
  private toggleExpanded(node: NodeId) {
    this.context.actions.onSelect(node);
    this.context.toggleExpanded(node);
    this.context.render();
  }

  private moveNode(
    drag: Extract<DragState, { kind: "node" }>,
    point: CanvasPosition,
  ) {
    const step = this.context.getDragStep();
    this.context.moveNode(drag.node, {
      x: drag.position.x + snapToStep(point.x - drag.start.x, step),
      y: drag.position.y + snapToStep(point.y - drag.start.y, step),
    });
  }

  private completeNodeSelection(start: CanvasPosition, end: CanvasPosition) {
    const bounds = normalizeRect(start, end);
    const scene = this.context.getScene();
    const selected = getCanvasNodes(this.context.getAnimations()).filter(
      (node) =>
        intersectsRect(
          bounds,
          getNodeBounds(
            node,
            scene.positions[node],
            scene.expanded.has(node),
            this.context.getPrototype(),
            this.context.getAnimations(),
            this.context.getReview(),
          ),
        ),
    );
    if (selected.length > 0) this.context.actions.onSelectNodes(selected);
    else this.context.actions.onClearSelection();
  }

  private completeFrameSelection(
    node: NodeId,
    start: CanvasPosition,
    end: CanvasPosition,
  ) {
    const bounds = normalizeRect(start, end);
    const position = this.context.getScene().positions[node];
    const indexes = Array.from(
      {
        length: getAnimatedSpriteFrameCount(
          node,
          this.context.getPrototype(),
          this.context.getAnimations(),
        ),
      },
      (_, index) => index,
    ).filter((index) =>
      intersectsRect(bounds, getFrameBounds(position, index)),
    );
    if (indexes.length > 0) this.context.actions.onSelectFrames(node, indexes);
  }

  private syncMarquee() {
    this.context.setMarquee(
      this.drag?.kind === "marquee" || this.drag?.kind === "frame-marquee"
        ? { start: this.drag.start, end: this.drag.end }
        : null,
    );
    this.context.render();
  }

  private screenPoint(event: PointerEvent): CanvasPosition {
    const bounds = this.canvas.getBoundingClientRect();
    return { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
  }

  private worldPoint(point: CanvasPosition): CanvasPosition {
    return this.context.viewport.toWorld(point);
  }
  private capture(event: PointerEvent) {
    this.canvas.setPointerCapture(event.pointerId);
  }
}
