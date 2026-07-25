import type { ProjectAsset } from "@/domain/asset";
import type { AssetKind } from "@/domain/asset";
import type {
  CharacterRecordContent,
  RecordContent,
  RecordContentForKind,
  SceneryRecordContent,
  SpriteSheetRecordContent,
  EditorCharacterAnimation,
  EditorSpriteSheetItem,
} from "@/domain/asset";
import { isRecordContentForAssetKind } from "@/domain/asset";

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

export function createDefaultRecord<K extends AssetKind>(
  kind: K,
  asset: ProjectAsset,
): RecordContentForKind<K> {
  const base = { prompt: asset.description };

  if (kind === "character" || kind === "object") {
    return {
      mode: "character",
      ...base,
      character: {
        prototypeName: `${asset.id}-prototype.png`,
        animations: structuredClone(defaultCharacterAnimations),
        nodePositions: {},
      },
    } as RecordContentForKind<K>;
  }

  if (kind === "scenery") {
    return {
      mode: "scenery",
      ...base,
      scenery: {
        layers: structuredClone(asset.scenery?.layers ?? []),
      },
    } as RecordContentForKind<K>;
  }

  return {
    mode: "sprite-sheet",
    ...base,
    spriteSheet: {
      gridSize: 8,
      items: structuredClone(kind === "tiles" ? tilesetItems : objectItems),
    },
  } as RecordContentForKind<K>;
}

export function mergeRecord<K extends AssetKind>(
  kind: K,
  fallback: RecordContentForKind<K>,
  saved: RecordContent | undefined,
): RecordContentForKind<K> {
  if (!saved || !isRecordContentForAssetKind(kind, saved)) return fallback;

  switch (fallback.mode) {
    case "character":
      return mergeCharacterRecord(
        fallback as CharacterRecordContent,
        saved as CharacterRecordContent,
      ) as RecordContentForKind<K>;
    case "scenery":
      return mergeSceneryRecord(
        saved as SceneryRecordContent,
      ) as RecordContentForKind<K>;
    case "sprite-sheet":
      return mergeSpriteSheetRecord(
        saved as SpriteSheetRecordContent,
      ) as RecordContentForKind<K>;
  }
}

function mergeCharacterRecord(
  fallback: CharacterRecordContent,
  saved: CharacterRecordContent,
): CharacterRecordContent {
  return {
    mode: "character",
    prompt: saved.prompt,
    character: {
      prototypeName:
        saved.character.prototypeName ??
        fallback.character.prototypeName ??
        "prototype.png",
      animations:
        saved.character.animations ?? fallback.character.animations ?? [],
      nodePositions: {
        ...fallback.character.nodePositions,
        ...saved.character.nodePositions,
      },
    },
  };
}

function mergeSceneryRecord(saved: SceneryRecordContent): SceneryRecordContent {
  return {
    mode: "scenery",
    prompt: saved.prompt,
    scenery: { layers: saved.scenery.layers },
  };
}

function mergeSpriteSheetRecord(
  saved: SpriteSheetRecordContent,
): SpriteSheetRecordContent {
  return {
    mode: "sprite-sheet",
    prompt: saved.prompt,
    spriteSheet: {
      gridSize: saved.spriteSheet.gridSize,
      items: saved.spriteSheet.items,
    },
  };
}
