import type { ProjectAsset } from "@/features/assets/domain";
import type { AssetKind } from "@/features/assets/domain";
import type {
  CharacterRecordContent,
  RecordContent,
  RecordContentForKind,
  SceneryRecordContent,
  SpriteSheetRecordContent,
  EditorCharacterAnimation,
  EditorSpriteSheetItem,
} from "@/features/assets/domain";
import { isRecordContentForAssetKind } from "@/features/assets/domain";

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

const tilesetAssetBase = "/assets/split_same_32px_grid_assets";

function createTilesetItem({
  id,
  label,
  imageUrl,
  tiles,
}: {
  id: string;
  label: string;
  imageUrl: string;
  tiles: EditorSpriteSheetItem["tiles"];
}): EditorSpriteSheetItem {
  return {
    id,
    label,
    imageUrl: `${tilesetAssetBase}/${imageUrl}`,
    tiles,
  };
}

const tilesetItems: EditorSpriteSheetItem[] = [
  createTilesetItem({
    id: "sofa-01",
    label: "Sofa 01",
    imageUrl: "sofas/sofa_01_96x64.png",
    tiles: [[0, 0], [1, 0], [2, 0], [0, 1], [1, 1], [2, 1]],
  }),
  createTilesetItem({
    id: "sofa-02",
    label: "Sofa 02",
    imageUrl: "sofas/sofa_02_96x64.png",
    tiles: [[3, 0], [4, 0], [5, 0], [3, 1], [4, 1], [5, 1]],
  }),
  createTilesetItem({
    id: "sofa-03",
    label: "Sofa 03",
    imageUrl: "sofas/sofa_03_96x64.png",
    tiles: [[0, 2], [1, 2], [2, 2], [0, 3], [1, 3], [2, 3]],
  }),
  createTilesetItem({
    id: "sofa-04",
    label: "Sofa 04",
    imageUrl: "sofas/sofa_04_96x64.png",
    tiles: [[3, 2], [4, 2], [5, 2], [3, 3], [4, 3], [5, 3]],
  }),
  createTilesetItem({
    id: "sofa-05",
    label: "Sofa 05",
    imageUrl: "sofas/sofa_05_96x64.png",
    tiles: [[0, 4], [1, 4], [2, 4], [0, 5], [1, 5], [2, 5]],
  }),
  ...Array.from({ length: 4 }, (_, index) =>
    createTilesetItem({
      id: `bed-${index + 1}`,
      label: `Bed ${String(index + 1).padStart(2, "0")}`,
      imageUrl: `beds/bed_${String(index + 1).padStart(2, "0")}_32x64.png`,
      tiles: [[6, index * 2], [6, index * 2 + 1]],
    }),
  ),
  ...["barrel_01", "barrel_02", "barrel_03", "barrel_04", "barrel_05", "jar_01", "jar_02", "jar_03"].map(
    (name, index) =>
      createTilesetItem({
        id: name.replace("_", "-"),
        label: name.replace("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase()),
        imageUrl: `items/${name}_32x32.png`,
        tiles: [[7, index]],
      }),
  ),
];

const objectItems: EditorSpriteSheetItem[] = [
  {
    id: "Object",
    label: "Object",
    tiles: [[3, 3]],
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
