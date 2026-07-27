import {
  CanvasSource,
  Container,
  Graphics,
  Texture,
  TilingSprite,
} from "pixi.js";

import type { CharacterCanvasModel } from "../CharacterCanvas.interface";
import { getCanvasNodes } from "../CharacterCanvas.constants";
import { getCharacterCanvasNodeId } from "../character-node";
import { normalizeBounds } from "../Interaction/CharacterStageGeometry";
import {
  getCharacterPixelScale,
  PIXEL_GRID_MAJOR_INTERVAL,
  STAGE_ACCENT,
  WORLD_HEIGHT,
  WORLD_WIDTH,
} from "../Runtime/CharacterStage.constants";
import type { CharacterSceneSnapshot } from "../Runtime/CharacterCanvas.types";
import { drawCharacterNode } from "./CharacterNodeRenderer";

export class CharacterStageRenderer {
  private readonly contentLayer = new Container();
  private readonly gridLayer = new Container();
  private gridPixelScale?: number;

  constructor(world: Container) {
    world.addChild(this.gridLayer, this.contentLayer);
  }

  render(state: CharacterSceneSnapshot, model: CharacterCanvasModel) {
    this.contentLayer
      .removeChildren()
      .forEach((child) => child.destroy({ children: true }));
    this.drawGrid(model.prototype);

    for (const node of getCanvasNodes(model.animations)) {
      this.contentLayer.addChild(
        drawCharacterNode({
          node,
          position: state.positions[node],
          selected: model.selection.nodeIds.some(
            (selectedNode) =>
              getCharacterCanvasNodeId(selectedNode, model.animations) === node,
          ),
          selectedFrames: model.selection.frames
            .filter(
              (frame) =>
                getCharacterCanvasNodeId(frame.nodeId, model.animations) ===
                node,
            )
            .map((frame) => frame.index),
          expanded: state.expanded.has(node),
          playing: state.playing.has(node),
          previewFrame: state.previewFrames.get(node) ?? 0,
          activeDirections: state.activeDirections,
          animations: model.animations,
          prototype: model.prototype,
        }),
      );
    }

    if (state.marquee) this.drawMarquee(state.marquee.start, state.marquee.end);
  }

  private drawGrid(prototype: CharacterCanvasModel["prototype"]) {
    const pixelScale = getCharacterPixelScale(prototype);
    if (this.gridPixelScale === pixelScale) return;
    this.gridPixelScale = pixelScale;
    this.gridLayer
      .removeChildren()
      .forEach((child) => child.destroy({ children: true }));

    this.gridLayer.addChild(createPixelGrid(pixelScale));
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

function createPixelGrid(pixelScale: number) {
  const canvas = document.createElement("canvas");
  canvas.width = PIXEL_GRID_MAJOR_INTERVAL;
  canvas.height = PIXEL_GRID_MAJOR_INTERVAL;
  const context = canvas.getContext("2d")!;

  for (let y = 0; y < canvas.height; y += 1) {
    for (let x = 0; x < canvas.width; x += 1) {
      context.fillStyle = (x + y) % 2 === 0 ? "#eeece7" : "#e8e5df";
      context.fillRect(x, y, 1, 1);
    }
  }

  const texture = new Texture({
    source: new CanvasSource({ resource: canvas }),
  });
  texture.source.scaleMode = "nearest";

  return new TilingSprite({
    texture,
    width: WORLD_WIDTH,
    height: WORLD_HEIGHT,
    tileScale: { x: pixelScale, y: pixelScale },
    roundPixels: true,
  });
}
