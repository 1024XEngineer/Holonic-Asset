import deepEqual from "fast-deep-equal";

import type { AssetRecord, SpriteAssetRecordData } from "./types";
import type { CharacterAnimation } from "../types";

export function describeAssetRecordChanges(
  previous: AssetRecord,
  current: AssetRecord,
): string {
  if (previous.mode !== current.mode) {
    return `Changed asset type from ${previous.mode} to ${current.mode}`;
  }

  if (previous.mode !== "character" && previous.mode !== "object") {
    return "Updated asset";
  }

  const before = getSprite(previous);
  const after = getSprite(current);
  const changes: string[] = [];

  if (!deepEqual(before.prototype, after.prototype)) {
    changes.push("Changed prototype");
  }

  const beforeAnimations = before.animations ?? [];
  const afterAnimations = after.animations ?? [];
  const beforeById = new Map(
    beforeAnimations.map((animation) => [animation.id, animation]),
  );
  const afterById = new Map(
    afterAnimations.map((animation) => [animation.id, animation]),
  );

  const added = afterAnimations.filter(
    (animation) => !beforeById.has(animation.id),
  );
  const removed = beforeAnimations.filter(
    (animation) => !afterById.has(animation.id),
  );
  if (added.length > 0) changes.push(formatAnimationChange("Added", added));
  if (removed.length > 0) {
    changes.push(formatAnimationChange("Removed", removed));
  }

  for (const animation of afterAnimations) {
    const previousAnimation = beforeById.get(animation.id);
    if (!previousAnimation) continue;
    if (previousAnimation.label !== animation.label) {
      changes.push(
        `Renamed animation ${previousAnimation.label} to ${animation.label}`,
      );
    }
    if (!sameAnimationContent(previousAnimation, animation)) {
      changes.push(`Changed animation: ${animation.label}`);
    }
  }

  if (previous.prompt !== current.prompt) changes.push("Changed prompt");

  return changes.length > 0 ? changes.join("; ") : "Updated asset";
}

function getSprite(record: AssetRecord): SpriteAssetRecordData {
  if (record.mode !== "character" && record.mode !== "object") {
    throw new Error("Sprite record required");
  }
  return record.mode === "character" ? record.character : record.object;
}

function formatAnimationChange(
  verb: "Added" | "Removed",
  animations: CharacterAnimation[],
) {
  const labels = animations.map((animation) => animation.label).join(", ");
  const noun = animations.length === 1 ? "animation" : "animations";
  return `${verb} ${animations.length} ${noun}: ${labels}`;
}

function sameAnimationContent(
  left: CharacterAnimation,
  right: CharacterAnimation,
) {
  const { label: _leftLabel, ...leftContent } = left;
  const { label: _rightLabel, ...rightContent } = right;
  return deepEqual(leftContent, rightContent);
}
