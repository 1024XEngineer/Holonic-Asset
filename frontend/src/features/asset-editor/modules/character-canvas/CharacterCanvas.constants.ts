import type { CharacterCanvasNodeId } from "./character-node";

export type CanvasPosition = {
  x: number;
  y: number;
};

export const CANVAS_NODES = [
  "prototype",
  "idle",
  "walk",
  "harvest",
  "jump",
  "celebrate",
] as const satisfies readonly CharacterCanvasNodeId[];

export const ANIMATION_NODES = new Set<CharacterCanvasNodeId>([
  "idle",
  "walk",
  "harvest",
  "jump",
  "celebrate",
]);

export const DEFAULT_CANVAS_POSITIONS: Record<
  (typeof CANVAS_NODES)[number],
  CanvasPosition
> = {
  prototype: { x: 160, y: 160 },
  idle: { x: 490, y: 160 },
  walk: { x: 900, y: 160 },
  harvest: { x: 1310, y: 160 },
  jump: { x: 520, y: 700 },
  celebrate: { x: 1060, y: 700 },
};
