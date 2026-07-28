import type {
  EditorCharacterAnimation,
  EditorCharacterAnimationClip,
  EditorCharacterAnimationGroup,
} from "../../domain";

export type AnimatedSpriteNodeId = string;

export type NodeId = AnimatedSpriteNodeId;

export type AnimatedSpriteDirectionMap =
  | ReadonlyMap<AnimatedSpriteNodeId, AnimatedSpriteNodeId>
  | Readonly<Record<string, AnimatedSpriteNodeId>>;

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

export function findAnimatedSpriteAnimation(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationClip | undefined {
  for (const animation of animations) {
    if (animation.kind === "group") {
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

export function findAnimatedSpriteAnimationGroup(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
): EditorCharacterAnimationGroup | undefined {
  const animation = animations.find((candidate) => candidate.id === node);
  if (animation?.kind === "group") return animation;
  return animations.find(
    (candidate) =>
      candidate.kind === "group" &&
      candidate.directions.some((direction) => direction.id === node),
  ) as EditorCharacterAnimationGroup | undefined;
}

export function getAnimatedSpriteAnimation(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
  directions?: AnimatedSpriteDirectionMap,
): EditorCharacterAnimationClip | undefined {
  const animation = animations.find((candidate) => candidate.id === node);
  if (!animation) return findAnimatedSpriteAnimation(node, animations);
  if (animation.kind !== "group") return animation;

  const activeDirectionId = getDirectionValue(directions, animation.id);
  return (
    animation.directions.find(
      (direction) => direction.id === activeDirectionId,
    ) ?? animation.directions[0]
  );
}

export function getPreferredAnimatedSpriteDirection(
  animation: EditorCharacterAnimationGroup,
  requested?: AnimatedSpriteNodeId,
  current?: AnimatedSpriteNodeId,
) {
  return (
    animation.directions.find((direction) => direction.id === requested) ??
    animation.directions.find((direction) => direction.id === current) ??
    animation.directions[0]
  );
}

export function getAnimatedSpriteNodeId(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
) {
  return findAnimatedSpriteAnimationGroup(node, animations)?.id ?? node;
}

export function getAnimatedSpriteDirectionId(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
  directions?: AnimatedSpriteDirectionMap,
) {
  return getAnimatedSpriteAnimation(node, animations, directions)?.id ?? node;
}

export function createDefaultAnimatedSpriteDirections(
  animations: EditorCharacterAnimation[],
) {
  return Object.fromEntries(
    animations
      .filter((animation) => animation.kind === "group")
      .map((animation) => [animation.id, animation.directions[0].id]),
  );
}

function getDirectionValue(
  directions: AnimatedSpriteDirectionMap | undefined,
  node: AnimatedSpriteNodeId,
) {
  if (!directions) return undefined;
  return directions instanceof Map
    ? directions.get(node)
    : (directions as Readonly<Record<string, AnimatedSpriteNodeId>>)[node];
}

export function getAnimatedSpriteNodeLabel(
  node: AnimatedSpriteNodeId,
  animations: EditorCharacterAnimation[],
  directions?: AnimatedSpriteDirectionMap,
) {
  const topLevelAnimation = animations.find(
    (candidate) => candidate.id === node,
  );
  if (topLevelAnimation?.kind === "group") {
    const direction = getAnimatedSpriteAnimation(node, animations, directions);
    return directions && direction
      ? `${topLevelAnimation.label} / ${direction.label}`
      : topLevelAnimation.label;
  }

  const animation = findAnimatedSpriteAnimation(node, animations);
  if (animation) {
    const parent = findAnimatedSpriteAnimationGroup(node, animations);
    return parent ? `${parent.label} / ${animation.label}` : animation.label;
  }

  return animatedSpriteNodeMeta[node]?.label ?? node;
}
