import type { NodeId } from "../character-node";
import {
  CANVAS_NODES,
  type CanvasPosition,
} from "../CharacterCanvas.constants";
import {
  getFrameBounds,
  getFrameCount,
  getNodeBounds,
  hitTestCharacterScene,
  intersects,
  normalizeBounds,
} from "./CharacterStageGeometry";
import { INITIAL_SCALE } from "../Runtime/CharacterStage.constants";
import type { CharacterStageContext } from "../Runtime/CharacterCanvas.types";

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

export class CharacterStageInteraction {
  private drag: DragState | null = null;
  private lastClick = { time: 0, x: 0, y: 0 };
  private readonly canvas: HTMLCanvasElement;
  private readonly context: CharacterStageContext;

  constructor(canvas: HTMLCanvasElement, context: CharacterStageContext) {
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
    const screen = this.screenPoint(event);
    const point = this.worldPoint(screen);
    if (event.button === 1) return;
    if (event.button !== 0) return;
    if (this.handleDoubleClick(screen)) return;

    const hit = this.hitTest(point);
    if (hit?.kind === "play") return this.togglePlaying(hit.node);
    if (hit?.kind === "expand") return this.toggleExpanded(hit.node);
    if (hit?.kind === "frame")
      return this.context.actions.onSelectFrame(hit.node, hit.index);
    if (hit?.kind === "frame-grid") {
      this.capture(event);
      this.drag = {
        kind: "frame-marquee",
        pointerId: event.pointerId,
        start: point,
        end: point,
        node: hit.node,
      };
    } else if (hit?.kind === "node") {
      this.capture(event);
      this.context.actions.onSelect(hit.node);
      const scene = this.context.getScene();
      this.drag = {
        kind: "node",
        pointerId: event.pointerId,
        start: point,
        node: hit.node,
        position: { ...scene.positions[hit.node] },
      };
    } else {
      this.capture(event);
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
    const screen = this.screenPoint(event);
    const point = this.worldPoint(screen);
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
    return hitTestCharacterScene(
      this.context.getScene(),
      point,
      this.context.getAnimations(),
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

  private handleDoubleClick(screen: CanvasPosition) {
    const now = performance.now();
    const isDouble =
      now - this.lastClick.time < 280 &&
      Math.hypot(screen.x - this.lastClick.x, screen.y - this.lastClick.y) < 5;
    this.lastClick = isDouble
      ? { time: 0, x: 0, y: 0 }
      : { time: now, x: screen.x, y: screen.y };
    if (!isDouble) return false;
    const focus = this.context.viewport.toWorld(screen);
    const scale = this.context.viewport.scale.x < 1 ? 1 : INITIAL_SCALE;
    this.context.viewport.setZoom(scale);
    this.context.viewport.moveCenter(focus);
    this.context.render();
    return true;
  }

  private moveNode(
    drag: Extract<DragState, { kind: "node" }>,
    point: CanvasPosition,
  ) {
    this.context.moveNode(drag.node, {
      x: drag.position.x + point.x - drag.start.x,
      y: drag.position.y + point.y - drag.start.y,
    });
  }

  private completeNodeSelection(start: CanvasPosition, end: CanvasPosition) {
    const bounds = normalizeBounds(start, end);
    const scene = this.context.getScene();
    const selected = CANVAS_NODES.filter((node) =>
      intersects(
        bounds,
        getNodeBounds(
          node,
          scene.positions[node],
          scene.expanded.has(node),
          this.context.getAnimations(),
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
    const bounds = normalizeBounds(start, end);
    const position = this.context.getScene().positions[node];
    const indexes = Array.from(
      { length: getFrameCount(node, this.context.getAnimations()) },
      (_, index) => index,
    ).filter((index) => intersects(bounds, getFrameBounds(position, index)));
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
