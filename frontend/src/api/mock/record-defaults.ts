import type { ProjectAsset } from "@/types/asset";
import type { AssetKind } from "@/types/asset-kind";
import type {
  RecordContent,
  EditorCharacterAnimation,
  EditorSpriteSheetItem,
} from "@/types/record";

const defaultCharacterAnimations: EditorCharacterAnimation[] = [
  {
    id: "idle",
    label: "Idle",
    frameCount: 6,
    audio: { label: "cloth_sway.wav", time: "0.06s" },
  },
  {
    id: "walk",
    label: "Walk",
    frameCount: 8,
    audio: { label: "footstep_grass.wav", time: "0.18s" },
  },
  {
    id: "harvest",
    label: "Harvest",
    frameCount: 12,
    audio: { label: "harvest_pickup.wav", time: "0.42s" },
  },
  { id: "jump", label: "Jump", frameCount: 10 },
  { id: "celebrate", label: "Celebrate", frameCount: 8 },
];

const tilesetItems: EditorSpriteSheetItem[] = [
  {
    id: "Bed",
    label: "Bed",
    icon: "bed",
    tiles: [
      { id: "bed-headboard", label: "Headboard", cells: [9] },
      { id: "bed-pillow", label: "Pillow", cells: [10] },
      { id: "bed-blanket", label: "Blanket", cells: [17] },
      { id: "bed-footboard", label: "Footboard", cells: [18] },
    ],
  },
  {
    id: "Street lamp",
    label: "Street lamp",
    icon: "lamp",
    tiles: [
      { id: "lamp-top", label: "Lamp top", cells: [5] },
      { id: "lamp-post", label: "Lamp post", cells: [13] },
      { id: "lamp-base", label: "Stone base", cells: [21] },
    ],
  },
  {
    id: "Street fence",
    label: "Street fence",
    icon: "fence",
    tiles: [
      { id: "fence-left", label: "Left cap", cells: [32] },
      { id: "fence-middle", label: "Fence middle", cells: [33, 34] },
      { id: "fence-right", label: "Right cap", cells: [35] },
      { id: "fence-corner", label: "Corner", cells: [36] },
    ],
  },
];

const objectItems: EditorSpriteSheetItem[] = [
  {
    id: "Object",
    label: "Object",
    icon: "object",
    tiles: [{ id: "object-base", label: "Base tile", cells: [27] }],
  },
];

export function createDefaultRecord(
  kind: AssetKind,
  asset: ProjectAsset,
): RecordContent {
  const base = { prompt: asset.description };

  if (kind === "character" || kind === "object") {
    return {
      ...base,
      character: {
        prototypeName: `${asset.id}-prototype.png`,
        animations: structuredClone(defaultCharacterAnimations),
        nodePositions: {},
      },
    };
  }

  if (kind === "scenery") {
    return {
      ...base,
      scenery: {
        layers: structuredClone(asset.scenery?.layers ?? []),
      },
    };
  }

  return {
    ...base,
    spriteSheet: {
      gridSize: 8,
      items: structuredClone(kind === "tiles" ? tilesetItems : objectItems),
    },
  };
}

export function mergeRecord(
  fallback: RecordContent,
  saved: RecordContent | undefined,
): RecordContent {
  if (!saved) return fallback;

  return {
    ...fallback,
    ...structuredClone(saved),
    character:
      fallback.character || saved.character
        ? {
            prototypeName:
              saved.character?.prototypeName ??
              fallback.character?.prototypeName ??
              "prototype.png",
            animations:
              saved.character?.animations ??
              fallback.character?.animations ??
              [],
            nodePositions: {
              ...fallback.character?.nodePositions,
              ...saved.character?.nodePositions,
            },
          }
        : undefined,
    scenery:
      fallback.scenery || saved.scenery
        ? {
            layers: saved.scenery?.layers ?? fallback.scenery?.layers ?? [],
          }
        : undefined,
    spriteSheet:
      fallback.spriteSheet || saved.spriteSheet
        ? {
            gridSize:
              saved.spriteSheet?.gridSize ??
              fallback.spriteSheet?.gridSize ??
              8,
            items:
              saved.spriteSheet?.items ?? fallback.spriteSheet?.items ?? [],
          }
        : undefined,
  };
}
