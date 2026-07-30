import type { EditorCharacterAnimation } from "@/features/asset-editor/types";

export type AnimatedSpriteNodeId = string;

export type NodeId = AnimatedSpriteNodeId;

export type AnimatedSpriteNodeMeta = {
  label: string;
  eyebrow: string;
};

export const animatedSpriteFrameColors = [
  "#f6c66e",
  "#f09b5b",
  "#91c7a5",
  "#7d9bd0",
  "#f2c17a",
  "#e68c67",
];

export const animatedSpriteNodeMeta: Record<string, AnimatedSpriteNodeMeta> = {
  prototype: { label: "Prototype", eyebrow: "Source" },
  metadata: { label: "Manifest", eyebrow: "Asset settings" },
};

export function getAnimatedSpriteAnimation(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
) {
  return animations.find((animation) => animation.id === node);
}

export function getAnimatedSpriteNodeLabel(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
) {
  return (
    getAnimatedSpriteAnimation(node, animations)?.label ??
    animatedSpriteNodeMeta[node]?.label ??
    node
  );
}
