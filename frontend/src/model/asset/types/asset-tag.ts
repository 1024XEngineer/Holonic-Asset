import { colord } from "colord";
import { z } from "zod";

export const defaultAssetTagColor = "#4F46E5";

export const assetTagSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Tag name is required.")
    .max(100, "Tag name must be at most 100 characters."),
  description: z
    .string()
    .trim()
    .max(255, "Tag description must be at most 255 characters.")
    .default(""),
  color: z
    .string()
    .trim()
    .regex(/^#[0-9a-f]{6}$/i, "Tag color must be a six-digit hex color."),
  tagId: z.number().int().positive().optional(),
  projectId: z.number().int().positive().optional(),
});

export type AssetTag = z.infer<typeof assetTagSchema>;

export function normalizeAssetTag(
  value: string | (Partial<AssetTag> & Pick<AssetTag, "name">),
): AssetTag | undefined {
  const name = typeof value === "string" ? value.trim() : value.name.trim();
  if (!name) return undefined;

  const description =
    typeof value === "string" ? "" : (value.description?.trim() ?? "");
  const color =
    typeof value === "string" ? defaultAssetTagColor : value.color?.trim();

  return {
    name,
    description,
    color:
      color && /^#[0-9a-f]{6}$/i.test(color) ? color : defaultAssetTagColor,
    ...(typeof value === "string" || value.tagId === undefined
      ? {}
      : { tagId: value.tagId }),
    ...(typeof value === "string" || value.projectId === undefined
      ? {}
      : { projectId: value.projectId }),
  };
}

export const presetAssetTagColors = [
  "#0969DA", // GitHub Blue
  "#1A7F37", // GitHub Green
  "#8250DF", // GitHub Purple
  "#CF222E", // GitHub Red
  "#BC4C00", // GitHub Orange
  "#D4A72C", // GitHub Yellow
  "#057A55", // GitHub Teal
  "#BF3989", // GitHub Pink
  "#4F46E5", // Indigo (Default)
  "#57606A", // Slate Gray
] as const;

export function hexToRgb(
  hex: string,
): { r: number; g: number; b: number } | null {
  const c = colord(hex);
  if (!c.isValid()) return null;
  const rgb = c.toRgb();
  return { r: rgb.r, g: rgb.g, b: rgb.b };
}

export function getRandomAssetTagColor(excludeColor?: string): string {
  const candidates = presetAssetTagColors.filter(
    (c) => c.toLowerCase() !== excludeColor?.toLowerCase(),
  );
  const randomIndex = Math.floor(Math.random() * candidates.length);
  return candidates[randomIndex] ?? defaultAssetTagColor;
}

export function getTagBadgeStyle(color?: string) {
  const hex =
    color && /^#[0-9a-f]{6}$/i.test(color) ? color : defaultAssetTagColor;
  const c = colord(hex);
  if (!c.isValid()) {
    return {
      backgroundColor: "var(--muted)",
      borderColor: "var(--border)",
    };
  }
  return {
    backgroundColor: c.alpha(0.12).toRgbString(),
    borderColor: c.alpha(0.35).toRgbString(),
  };
}

export function mergeAssetTags(
  ...collections: ReadonlyArray<
    readonly (string | (Partial<AssetTag> & Pick<AssetTag, "name">))[]
  >
): AssetTag[] {
  const tags = new Map<string, AssetTag>();

  for (const collection of collections) {
    for (const value of collection) {
      const tag = normalizeAssetTag(value);
      if (!tag) continue;

      const key = tag.name.toLocaleLowerCase();
      const existing = tags.get(key);
      if (!existing) {
        tags.set(key, tag);
        continue;
      }

      tags.set(key, {
        name: existing.name,
        description: existing.description || tag.description,
        color:
          existing.color === defaultAssetTagColor &&
          tag.color !== defaultAssetTagColor
            ? tag.color
            : existing.color,
        ...mergeTagIdentity(existing, tag),
      });
    }
  }

  return [...tags.values()];
}

function mergeTagIdentity(existing: AssetTag, next: AssetTag) {
  const identity: Pick<AssetTag, "tagId" | "projectId"> = {};
  if (existing.tagId !== undefined) identity.tagId = existing.tagId;
  else if (next.tagId !== undefined) identity.tagId = next.tagId;
  if (existing.projectId !== undefined) identity.projectId = existing.projectId;
  else if (next.projectId !== undefined) identity.projectId = next.projectId;
  return identity;
}
