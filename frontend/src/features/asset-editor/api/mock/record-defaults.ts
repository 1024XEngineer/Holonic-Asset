import type { AssetKind, ProjectAsset } from "@/features/assets/domain";
import type {
  CharacterEditorDocument,
  EditorCharacterAnimation,
  EditorSpriteSheetItem,
  EditorDocumentForKind,
  SceneryEditorDocument,
  SpriteSheetEditorDocument,
} from "../../domain";
import { isEditorDocumentForAssetKind } from "../../domain";

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

export function createDefaultEditorDocument<K extends AssetKind>(
  kind: K,
  asset: ProjectAsset,
): EditorDocumentForKind<K> {
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
    } as EditorDocumentForKind<K>;
  }

  if (kind === "scenery") {
    return {
      mode: "scenery",
      ...base,
      scenery: {
        layers: structuredClone(asset.scenery?.layers ?? []),
      },
    } as EditorDocumentForKind<K>;
  }

  return {
    mode: "sprite-sheet",
    ...base,
    spriteSheet: {
      gridSize: 8,
      items: structuredClone(kind === "tiles" ? tilesetItems : objectItems),
    },
  } as EditorDocumentForKind<K>;
}

export function mergeEditorDocument<K extends AssetKind>(
  kind: K,
  fallback: EditorDocumentForKind<K>,
  saved: unknown,
): EditorDocumentForKind<K> {
  if (!saved || !isEditorDocumentForAssetKind(kind, saved)) return fallback;

  switch (fallback.mode) {
    case "character":
      return mergeCharacterRecord(
        fallback as CharacterEditorDocument,
        saved as CharacterEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "scenery":
      return mergeSceneryRecord(
        saved as SceneryEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "sprite-sheet":
      return mergeSpriteSheetRecord(
        saved as SpriteSheetEditorDocument,
      ) as EditorDocumentForKind<K>;
  }
}

function mergeCharacterRecord(
  fallback: CharacterEditorDocument,
  saved: CharacterEditorDocument,
): CharacterEditorDocument {
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

function mergeSceneryRecord(
  saved: SceneryEditorDocument,
): SceneryEditorDocument {
  return {
    mode: "scenery",
    prompt: saved.prompt,
    scenery: { layers: saved.scenery.layers },
  };
}

function mergeSpriteSheetRecord(
  saved: SpriteSheetEditorDocument,
): SpriteSheetEditorDocument {
  return {
    mode: "sprite-sheet",
    prompt: saved.prompt,
    spriteSheet: {
      gridSize: saved.spriteSheet.gridSize,
      items: saved.spriteSheet.items,
    },
  };
}
