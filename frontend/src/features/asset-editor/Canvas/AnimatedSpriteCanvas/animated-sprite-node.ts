import type { CharacterAnimation } from "@/model";

export type AnimatedSpriteNodeId = string;
export type NodeId = AnimatedSpriteNodeId;

export const animatedSpriteFrameColors = [
  "#f6c66e",
  "#f09b5b",
  "#91c7a5",
  "#7d9bd0",
  "#f2c17a",
  "#e68c67",
];

export function getAnimatedSpriteAnimation(
  node: AnimatedSpriteNodeId,
  animations: readonly CharacterAnimation[],
) {
  return animations.find((animation) => animation.id === node);
}

export function getAnimatedSpriteNodeLabel(
  node: AnimatedSpriteNodeId,
  animations: readonly CharacterAnimation[],
) {
  return node === "prototype"
    ? "Prototype"
    : (getAnimatedSpriteAnimation(node, animations)?.label ?? node);
}
