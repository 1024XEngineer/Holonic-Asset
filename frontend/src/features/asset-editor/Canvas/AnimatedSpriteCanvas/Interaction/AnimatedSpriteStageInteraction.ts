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
import type { AnimatedSpriteCanvasFrameSelection } from "../AnimatedSpriteCanvas.interface";

const MARQUEE_DRAG_THRESHOLD = 3;

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
      additive: boolean;
      startTarget:
        | { kind: "frame"; node: NodeId; index: number }
        | { kind: "frame-grid" }
        | { kind: "empty" };
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
    canvas.addEventListener("pointerup", this.onPointerUp);
    canvas.addEventListener("pointercancel", this.onPointerCancel);
    canvas.addEventListener("lostpointercapture", this.onLostPointerCapture);
    canvas.addEventListener("contextmenu", this.onContextMenu);
  }

  destroy() {
    this.canvas.removeEventListener("pointerdown", this.onPointerDown);
    this.canvas.removeEventListener("pointermove", this.onPointerMove);
    this.canvas.removeEventListener("pointerup", this.onPointerUp);
    this.canvas.removeEventListener("pointercancel", this.onPointerCancel);
    this.canvas.removeEventListener(
      "lostpointercapture",
      this.onLostPointerCapture,
    );
    this.canvas.removeEventListener("contextmenu", this.onContextMenu);
    if (this.drag) this.cancelPointer(this.drag.pointerId);
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
    const additive = event.ctrlKey || event.metaKey;
    this.capture(event);
    if (hit?.kind === "frame" || hit?.kind === "frame-grid") {
      this.drag = {
        kind: "marquee",
        pointerId: event.pointerId,
        start: point,
        end: point,
        additive,
        startTarget:
          hit.kind === "frame"
            ? { kind: "frame", node: hit.node, index: hit.index }
            : { kind: "frame-grid" },
      };
    } else if (hit?.kind === "node") {
      this.context.actions.onSelect(hit.node, additive);
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
        additive,
        startTarget: { kind: "empty" },
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

  private onPointerUp = (event: PointerEvent) => {
    if (!this.drag || this.drag.pointerId !== event.pointerId) return;
    const completed = this.drag;
    const completedNodePosition =
      completed.kind === "node"
        ? { ...this.context.getScene().positions[completed.node] }
        : null;
    this.clearPointer(event.pointerId);
    if (completed.kind === "marquee") this.completeMarquee(completed);
    if (completed.kind === "node" && completedNodePosition)
      this.context.actions.onNodePositionChange(
        completed.node,
        completedNodePosition,
      );
  };

  private onPointerCancel = (event: PointerEvent) => {
    this.cancelPointer(event.pointerId);
  };

  private onLostPointerCapture = (event: PointerEvent) => {
    this.cancelPointer(event.pointerId, false);
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

  private completeMarquee(drag: Extract<DragState, { kind: "marquee" }>) {
    if (!this.hasMarqueeDrag(drag)) {
      if (drag.startTarget.kind === "frame") {
        this.context.actions.onSelectFrame(
          drag.startTarget.node,
          drag.startTarget.index,
          drag.additive,
        );
      } else if (drag.startTarget.kind === "empty" && !drag.additive) {
        this.context.actions.onClearSelection();
      }
      return;
    }

    const bounds = normalizeRect(drag.start, drag.end);
    const scene = this.context.getScene();
    const nodes = getCanvasNodes(this.context.getAnimations());
    const frames: AnimatedSpriteCanvasFrameSelection[] = nodes.flatMap(
      (node) =>
        scene.expanded.has(node)
          ? Array.from(
              {
                length: getAnimatedSpriteFrameCount(
                  node,
                  this.context.getPrototype(),
                  this.context.getAnimations(),
                ),
              },
              (_, index) => ({ nodeId: node, index }),
            ).filter(({ index }) =>
              intersectsRect(
                bounds,
                getFrameBounds(scene.positions[node], index),
              ),
            )
          : [],
    );
    if (frames.length > 0) {
      this.context.actions.onSelectFrames(frames, drag.additive);
      return;
    }

    const selectedNodes = nodes.filter((node) =>
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
    if (selectedNodes.length > 0)
      this.context.actions.onSelectNodes(selectedNodes, drag.additive);
    else if (!drag.additive) this.context.actions.onClearSelection();
  }

  private hasMarqueeDrag(drag: Extract<DragState, { kind: "marquee" }>) {
    return (
      Math.max(
        Math.abs(drag.end.x - drag.start.x),
        Math.abs(drag.end.y - drag.start.y),
      ) >= MARQUEE_DRAG_THRESHOLD
    );
  }

  private cancelPointer(pointerId: number, releaseCapture = true) {
    if (!this.drag || this.drag.pointerId !== pointerId) return;
    if (this.drag.kind === "node")
      this.context.moveNode(this.drag.node, { ...this.drag.position });
    this.clearPointer(pointerId, releaseCapture);
  }

  private clearPointer(pointerId: number, releaseCapture = true) {
    this.drag = null;
    if (releaseCapture && this.canvas.hasPointerCapture(pointerId))
      this.canvas.releasePointerCapture(pointerId);
    this.syncMarquee();
  }

  private syncMarquee() {
    this.context.setMarquee(
      this.drag?.kind === "marquee"
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
