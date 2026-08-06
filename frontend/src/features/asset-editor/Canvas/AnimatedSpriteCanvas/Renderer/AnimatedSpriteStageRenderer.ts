import {
  CanvasSource,
  Container,
  Graphics,
  Texture,
  TilingSprite,
} from "pixi.js";
import type { Viewport } from "pixi-viewport";
import type { AnimatedSpriteCanvasModel } from "../AnimatedSpriteCanvas.interface";
import { getCanvasNodes } from "../AnimatedSpriteCanvas.constants";
import { normalizeBounds } from "../Interaction/AnimatedSpriteStageGeometry";
import {
  getAnimatedSpritePixelScale,
  PIXEL_GRID_MAJOR_INTERVAL,
  STAGE_ACCENT,
} from "../Runtime/AnimatedSpriteStage.constants";
import type { AnimatedSpriteSceneSnapshot } from "../Runtime/AnimatedSpriteCanvas.types";
import { drawAnimatedSpriteNode } from "./AnimatedSpriteNodeRenderer";

export class AnimatedSpriteStageRenderer {
  private readonly contentLayer = new Container();
  private readonly gridLayer = new Container();
  private grid?: TilingSprite;
  private gridPixelScale?: number;

  constructor(stage: Container, world: Container) {
    this.gridLayer.eventMode = "none";
    stage.addChildAt(this.gridLayer, 0);
    world.addChild(this.contentLayer);
  }

  render(state: AnimatedSpriteSceneSnapshot, model: AnimatedSpriteCanvasModel) {
    this.contentLayer
      .removeChildren()
      .forEach((child) => child.destroy({ children: true }));
    this.drawGrid(model.prototype);
    for (const node of getCanvasNodes(model.animations)) {
      this.contentLayer.addChild(
        drawAnimatedSpriteNode({
          node,
          position: state.positions[node],
          selected: model.selection.nodeIds.includes(node),
          selectedFrames: model.selection.frames
            .filter((frame) => frame.nodeId === node)
            .map((frame) => frame.index),
          expanded: state.expanded.has(node),
          playing: state.playing.has(node),
          previewFrame: state.previewFrames.get(node) ?? 0,
          animations: model.animations,
          prototype: model.prototype,
          unavailableTextureUrls: model.unavailableTextureUrls,
        }),
      );
    }
    if (state.marquee) this.drawMarquee(state.marquee.start, state.marquee.end);
  }

  syncViewport(
    viewport: Viewport,
    prototype: AnimatedSpriteCanvasModel["prototype"],
  ) {
    this.drawGrid(prototype);
    if (!this.grid) return;
    this.grid.setSize(viewport.screenWidth, viewport.screenHeight);
    this.grid.tileScale.set(
      getAnimatedSpritePixelScale(prototype) * viewport.scale.x,
    );
    this.grid.tilePosition.set(viewport.x, viewport.y);
  }

  private drawGrid(prototype: AnimatedSpriteCanvasModel["prototype"]) {
    const pixelScale = getAnimatedSpritePixelScale(prototype);
    if (this.gridPixelScale === pixelScale && this.grid) return;
    this.gridPixelScale = pixelScale;
    this.grid?.destroy({ texture: true, textureSource: true });
    this.grid = createPixelGrid();
    this.gridLayer.removeChildren();
    this.gridLayer.addChild(this.grid);
  }

  private drawMarquee(
    start: { x: number; y: number },
    end: { x: number; y: number },
  ) {
    const bounds = normalizeBounds(start, end);
    this.contentLayer.addChild(
      new Graphics()
        .rect(bounds.x, bounds.y, bounds.width, bounds.height)
        .fill({ color: STAGE_ACCENT, alpha: 0.1 })
        .stroke({ color: STAGE_ACCENT, width: 1 }),
    );
  }
}

function createPixelGrid() {
  const canvas = document.createElement("canvas");
  canvas.width = PIXEL_GRID_MAJOR_INTERVAL;
  canvas.height = PIXEL_GRID_MAJOR_INTERVAL;
  const context = canvas.getContext("2d");
  if (!context) throw new Error("Canvas 2D context is unavailable.");
  for (let y = 0; y < canvas.height; y += 1)
    for (let x = 0; x < canvas.width; x += 1) {
      context.fillStyle = (x + y) % 2 === 0 ? "#eeece7" : "#e8e5df";
      context.fillRect(x, y, 1, 1);
    }
  const texture = new Texture({
    source: new CanvasSource({ resource: canvas }),
  });
  texture.source.scaleMode = "nearest";
  return new TilingSprite({
    texture,
    width: 1,
    height: 1,
    tileScale: { x: 1, y: 1 },
    roundPixels: true,
  });
}
