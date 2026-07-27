import {
  isEditorCharacterAnimationGroup,
  type EditorCharacterAnimation,
  type EditorCharacterAnimationClip,
  type EditorCharacterAnimationGroup,
} from "../../domain";

export type CharacterCanvasNodeId = string;

export type NodeId = CharacterCanvasNodeId;

export type CharacterDirectionMap =
  | ReadonlyMap<CharacterCanvasNodeId, CharacterCanvasNodeId>
  | Readonly<Record<string, CharacterCanvasNodeId>>;

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

export const characterNodeMeta: Record<string, CharacterCanvasNodeMeta> = {
  prototype: { label: "Prototype", eyebrow: "Source" },
  metadata: { label: "Manifest", eyebrow: "Asset settings" },
};

export function findCharacterAnimation(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationClip | undefined {
  for (const animation of animations) {
    if (isEditorCharacterAnimationGroup(animation)) {
      const direction = animation.directions.find(
        (candidate) => candidate.id === node,
      );
      if (direction) return direction;
    } else if (animation.id === node) {
      return animation;
    }
  }
  return undefined;
}

export function findCharacterAnimationGroup(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationGroup | undefined {
  const animation = animations.find((candidate) => candidate.id === node);
  if (animation && isEditorCharacterAnimationGroup(animation)) return animation;
  return animations.find(
    (candidate) =>
      isEditorCharacterAnimationGroup(candidate) &&
      candidate.directions.some((direction) => direction.id === node),
  ) as EditorCharacterAnimationGroup | undefined;
}

export function getCharacterCanvasAnimation(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
  directions?: CharacterDirectionMap,
): EditorCharacterAnimationClip | undefined {
  const animation = animations.find((candidate) => candidate.id === node);
  if (!animation) return findCharacterAnimation(node, animations);
  if (!isEditorCharacterAnimationGroup(animation)) return animation;

  const activeDirectionId = getDirectionValue(directions, animation.id);
  return (
    animation.directions.find(
      (direction) => direction.id === activeDirectionId,
    ) ?? animation.directions[0]
  );
}

export function getPreferredCharacterDirection(
  animation: EditorCharacterAnimationGroup,
  requested?: CharacterCanvasNodeId,
  current?: CharacterCanvasNodeId,
) {
  return (
    animation.directions.find((direction) => direction.id === requested) ??
    animation.directions.find((direction) => direction.id === current) ??
    animation.directions[0]
  );
}

export function getCharacterCanvasNodeId(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
) {
  return findCharacterAnimationGroup(node, animations)?.id ?? node;
}

export function getCharacterDirectionId(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
  directions?: CharacterDirectionMap,
) {
  return getCharacterCanvasAnimation(node, animations, directions)?.id ?? node;
}

export function createDefaultCharacterDirections(
  animations: EditorCharacterAnimation[],
) {
  return Object.fromEntries(
    animations
      .filter(isEditorCharacterAnimationGroup)
      .map((animation) => [animation.id, animation.directions[0].id]),
  );
}

function getDirectionValue(
  directions: CharacterDirectionMap | undefined,
  node: CharacterCanvasNodeId,
) {
  if (!directions) return undefined;
  return directions instanceof Map
    ? directions.get(node)
    : (directions as Readonly<Record<string, CharacterCanvasNodeId>>)[node];
}

export function getCharacterNodeLabel(
  node: CharacterCanvasNodeId,
  animations: EditorCharacterAnimation[],
  directions?: CharacterDirectionMap,
) {
  const topLevelAnimation = animations.find(
    (candidate) => candidate.id === node,
  );
  if (topLevelAnimation && isEditorCharacterAnimationGroup(topLevelAnimation)) {
    const direction = getCharacterCanvasAnimation(node, animations, directions);
    return directions && direction
      ? `${topLevelAnimation.label} / ${direction.label}`
      : topLevelAnimation.label;
  }

  const animation = findCharacterAnimation(node, animations);
  if (animation) {
    const parent = findCharacterAnimationGroup(node, animations);
    return parent ? `${parent.label} / ${animation.label}` : animation.label;
  }

  return characterNodeMeta[node]?.label ?? node;
}
