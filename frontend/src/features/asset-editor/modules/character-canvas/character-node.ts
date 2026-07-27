import type { EditorCharacterAnimation } from "@/features/assets/domain";

export type CharacterCanvasNodeId =
  | "prototype"
  | "idle"
  | "walk"
  | "harvest"
  | "jump"
  | "celebrate"
  | "metadata";

export type NodeId = CharacterCanvasNodeId;

export type CharacterCanvasNodeMeta = {
  label: string;
  eyebrow: string;
};

export const characterFrameColors = [
  "#f6c66e",
  "#f09b5b",
  "#91c7a5",
  "#7d9bd0",
  "#f2c17a",
  "#e68c67",
];

export const characterNodeMeta: Record<
  CharacterCanvasNodeId,
  CharacterCanvasNodeMeta
> = {
  prototype: { label: "Prototype", eyebrow: "Source" },
  idle: { label: "Idle", eyebrow: "Animation" },
  walk: { label: "Walk", eyebrow: "Animation" },
  harvest: { label: "Harvest", eyebrow: "Animation" },
  jump: { label: "Jump", eyebrow: "Animation" },
  celebrate: { label: "Celebrate", eyebrow: "Animation" },
  metadata: { label: "Manifest", eyebrow: "Asset settings" },
};

export function findCharacterAnimation(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
) {
  return animations.find((animation) => animation.id === node);
}

export function getCharacterNodeLabel(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
) {
  return (
    findCharacterAnimation(node, animations)?.label ??
    characterNodeMeta[node].label
  );
}
