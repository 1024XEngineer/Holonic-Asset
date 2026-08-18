type JsonObject = Record<string, unknown>;

const collectionIdentityKeys = {
  animations: "id",
  components: "id",
  items: "name",
  layers: "id",
} as const;

export function mergeAssetContentPatch(
  current: unknown,
  patch: unknown,
): JsonObject {
  const currentContent = asJsonObject(current, "current asset content");
  const patchContent = asJsonObject(patch, "generation result content");
  const merged: JsonObject = { ...currentContent, ...patchContent };

  for (const [field, identityKey] of Object.entries(collectionIdentityKeys)) {
    if (!(field in patchContent)) continue;
    merged[field] =
      field === "animations"
        ? mergeAnimationCollection(currentContent[field], patchContent[field])
        : mergeCollection(
            currentContent[field],
            patchContent[field],
            identityKey,
          );
  }

  return merged;
}

function mergeAnimationCollection(current: unknown, patch: unknown) {
  if (!Array.isArray(patch)) return patch;

  const currentItems = Array.isArray(current) ? current : [];
  const merged = currentItems.map((value) => cloneValue(value));
  const usedIds = new Set(
    [...currentItems, ...patch].flatMap((value) => {
      const item = asOptionalJsonObject(value);
      return item && isPositiveInteger(item.id) ? [String(item.id)] : [];
    }),
  );
  let nextId =
    Math.max(
      0,
      ...currentItems.flatMap((value) => {
        const item = asOptionalJsonObject(value);
        return item && isPositiveInteger(item.id) ? [Number(item.id)] : [];
      }),
    ) + 1;

  for (const patchValue of patch) {
    const patchItem = asOptionalJsonObject(patchValue);
    if (!patchItem) {
      merged.push(cloneValue(patchValue));
      continue;
    }

    let normalizedPatch = patchItem;
    if (!isPositiveInteger(patchItem.id)) {
      while (usedIds.has(String(nextId))) nextId += 1;
      normalizedPatch = { ...patchItem, id: nextId };
      usedIds.add(String(nextId));
      nextId += 1;
    }

    const identity = normalizedPatch.id;
    const existingIndex = merged.findIndex((value) => {
      const currentItem = asOptionalJsonObject(value);
      return (
        currentItem?.id !== undefined &&
        String(currentItem.id) === String(identity)
      );
    });

    if (existingIndex === -1) {
      merged.push(cloneValue(normalizedPatch));
      continue;
    }
    const currentItem = asOptionalJsonObject(merged[existingIndex]);
    merged[existingIndex] = currentItem
      ? { ...currentItem, ...normalizedPatch }
      : cloneValue(normalizedPatch);
  }
  return merged;
}

function isPositiveInteger(value: unknown): value is number | string {
  if (typeof value !== "number" && typeof value !== "string") return false;
  const numeric = Number(value);
  return Number.isSafeInteger(numeric) && numeric > 0;
}

function mergeCollection(
  current: unknown,
  patch: unknown,
  identityKey: string,
) {
  if (!Array.isArray(patch)) return patch;
  if (!Array.isArray(current)) return patch;

  const merged = current.map((value) => cloneValue(value));
  for (const patchValue of patch) {
    const patchItem = asOptionalJsonObject(patchValue);
    const identity = patchItem?.[identityKey];
    const existingIndex = merged.findIndex((value) => {
      const currentItem = asOptionalJsonObject(value);
      return (
        identity !== undefined &&
        currentItem?.[identityKey] !== undefined &&
        String(currentItem[identityKey]) === String(identity)
      );
    });

    if (existingIndex === -1) {
      merged.push(cloneValue(patchValue));
      continue;
    }
    const currentItem = asOptionalJsonObject(merged[existingIndex]);
    merged[existingIndex] =
      currentItem && patchItem
        ? { ...currentItem, ...patchItem }
        : cloneValue(patchValue);
  }
  return merged;
}

function asJsonObject(value: unknown, label: string): JsonObject {
  const object = asOptionalJsonObject(value);
  if (!object) throw new Error(`${label} must be an object.`);
  return object;
}

function asOptionalJsonObject(value: unknown): JsonObject | undefined {
  return value !== null && typeof value === "object" && !Array.isArray(value)
    ? (value as JsonObject)
    : undefined;
}

function cloneValue<T>(value: T): T {
  return value === undefined ? value : structuredClone(value);
}
