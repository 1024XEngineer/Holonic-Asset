import type {
  AssetAnimationResponse,
  AssetImageResourceResponse,
  CoreSpriteAssetContent,
  CoreSpriteAssetContentPatch,
} from "./asset.contract";

export function mergeAssetContentPatch(
  current: CoreSpriteAssetContent,
  patch: CoreSpriteAssetContentPatch,
): CoreSpriteAssetContent {
  const merged: CoreSpriteAssetContent = {
    ...current,
    ...(patch.directionCount === undefined
      ? {}
      : { directionCount: patch.directionCount }),
    ...(patch.metadata === undefined ? {} : { metadata: patch.metadata }),
  };

  if (patch.prototype) {
    merged.prototype = mergeCollection(current.prototype, patch.prototype);
  }
  if (patch.animations) {
    merged.animations = mergeAnimationCollection(
      current.animations ?? [],
      patch.animations,
    );
  }

  return merged;
}

type AnimationPatch = Omit<AssetAnimationResponse, "id"> & { id?: number };

function mergeAnimationCollection(
  current: AssetAnimationResponse[],
  patch: AnimationPatch[],
): AssetAnimationResponse[] {
  const merged = current.map((value) => structuredClone(value));
  const usedIds = new Set(
    [...current, ...patch].flatMap((value) =>
      isPositiveInteger(value.id) ? [value.id] : [],
    ),
  );
  let nextId = Math.max(0, ...current.map(({ id }) => id)) + 1;

  for (const patchItem of patch) {
    const id = patchItem.id ?? nextAvailableId(usedIds, nextId);
    const normalizedPatch: AssetAnimationResponse = {
      ...patchItem,
      id,
    };
    nextId = id + 1;
    usedIds.add(id);

    const existingIndex = merged.findIndex(
      ({ id: currentId }) => currentId === id,
    );
    if (existingIndex === -1) {
      merged.push(normalizedPatch);
      continue;
    }
    merged[existingIndex] = {
      ...merged[existingIndex],
      ...normalizedPatch,
    };
  }
  return merged;
}

function nextAvailableId(usedIds: Set<number>, start: number) {
  let id = start;
  while (usedIds.has(id)) id += 1;
  return id;
}

function mergeCollection(
  current: AssetImageResourceResponse[],
  patch: AssetImageResourceResponse[],
): AssetImageResourceResponse[] {
  const merged = current.map((value) => structuredClone(value));
  for (const patchItem of patch) {
    const existingIndex = merged.findIndex(({ id }) => id === patchItem.id);
    if (existingIndex === -1) {
      merged.push(structuredClone(patchItem));
      continue;
    }
    merged[existingIndex] = {
      ...merged[existingIndex],
      ...patchItem,
    };
  }
  return merged;
}

function isPositiveInteger(value: number | undefined): value is number {
  return value !== undefined && Number.isSafeInteger(value) && value > 0;
}
