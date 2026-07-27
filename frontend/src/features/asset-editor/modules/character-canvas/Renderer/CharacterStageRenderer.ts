import { Container, Graphics } from "pixi.js";

import type { CharacterCanvasModel } from "../CharacterCanvas.interface";
import { CANVAS_NODES } from "../CharacterCanvas.constants";
import { normalizeBounds } from "../Interaction/CharacterStageGeometry";
import {
  STAGE_ACCENT,
  STAGE_BACKGROUND,
  WORLD_HEIGHT,
  WORLD_WIDTH,
} from "../Runtime/CharacterStage.constants";
import type { CharacterSceneSnapshot } from "../Runtime/CharacterCanvas.types";
import { drawCharacterNode } from "./CharacterNodeRenderer";

export class CharacterStageRenderer {
  private readonly world: Container;

  constructor(world: Container) {
    this.world = world;
  }

  render(state: CharacterSceneSnapshot, model: CharacterCanvasModel) {
    this.world
      .removeChildren()
      .forEach((child) => child.destroy({ children: true }));
    this.drawGrid();

    for (const node of CANVAS_NODES) {
      this.world.addChild(
        drawCharacterNode({
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
        }),
      );
    }

    if (state.marquee) this.drawMarquee(state.marquee.start, state.marquee.end);
  }

  private drawGrid() {
    const grid = new Graphics()
      .rect(0, 0, WORLD_WIDTH, WORLD_HEIGHT)
      .fill(STAGE_BACKGROUND);
    for (let x = 0; x <= WORLD_WIDTH; x += 32)
      grid.moveTo(x, 0).lineTo(x, WORLD_HEIGHT);
    for (let y = 0; y <= WORLD_HEIGHT; y += 32)
      grid.moveTo(0, y).lineTo(WORLD_WIDTH, y);
    grid.stroke({ color: 0x342a20, alpha: 0.08, width: 1 });
    this.world.addChild(grid);
  }

  private drawMarquee(
    start: { x: number; y: number },
    end: { x: number; y: number },
  ) {
    const bounds = normalizeBounds(start, end);
    this.world.addChild(
      new Graphics()
        .rect(bounds.x, bounds.y, bounds.width, bounds.height)
        .fill({ color: STAGE_ACCENT, alpha: 0.1 })
        .stroke({ color: STAGE_ACCENT, width: 1 }),
    );
  }
}
