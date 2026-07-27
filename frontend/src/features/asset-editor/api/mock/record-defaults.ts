import type { AssetKind, ProjectAsset } from "@/features/assets/domain";
import type {
  CharacterEditorDocument,
  EditorCharacterAnimation,
  EditorCharacterAnimationClip,
  EditorCharacterAnimationGroup,
  EditorCharacterSpriteSheet,
  EditorSpriteSheetItem,
  EditorDocumentForKind,
  SceneryEditorDocument,
  SpriteSheetEditorDocument,
} from "../../domain";
import { isEditorDocumentForAssetKind } from "../../domain";

const swordsmanPrototype: EditorCharacterSpriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "/assets/characters/swordsman/prototype.png",
  frameWidth: 64,
  frameHeight: 64,
  columns: 4,
  rows: 1,
};

const knightPrototype: EditorCharacterSpriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "/assets/characters/knight/prototype.png",
  frameWidth: 128,
  frameHeight: 128,
  columns: 1,
  rows: 1,
};

function createPngAnimation(
  id: string,
  label: string,
  imageUrl: string,
  frameCount: number,
  frameWidth: number,
  frameHeight: number,
): EditorCharacterAnimationClip {
  return {
    id,
    label,
    frameCount,
    spriteSheet: {
      format: "png-sprite-sheet",
      imageUrl,
      frameWidth,
      frameHeight,
      columns: frameCount,
      rows: 1,
    },
  };
}

const swordsmanDirections = [
  { id: "front", label: "Front" },
  { id: "back", label: "Back" },
  { id: "left", label: "Left" },
  { id: "right", label: "Right" },
] as const;

function createSwordsmanAnimation(
  id: string,
  label: string,
  frameCounts: Record<(typeof swordsmanDirections)[number]["id"], number>,
): EditorCharacterAnimationGroup {
  return {
    id,
    label,
    directions: swordsmanDirections.map((direction) =>
      createPngAnimation(
        `${id}/${direction.id}`,
        direction.label,
        `/assets/characters/swordsman/${id}/${direction.id}.png`,
        frameCounts[direction.id],
        64,
        64,
      ),
    ),
  };
}

const characterAnimationsByAssetId: Record<string, EditorCharacterAnimation[]> =
  {
    swordsman: [
      createSwordsmanAnimation("idle", "Idle", {
        front: 12,
        back: 4,
        left: 12,
        right: 12,
      }),
      createSwordsmanAnimation("attack", "Attack", {
        front: 8,
        back: 8,
        left: 8,
        right: 8,
      }),
    ],
    knight: [
      createPngAnimation(
        "idle",
        "Idle",
        "/assets/characters/knight/idle.png",
        4,
        128,
        128,
      ),
      createPngAnimation(
        "attack",
        "Attack",
        "/assets/characters/knight/attack-1.png",
        5,
        128,
        128,
      ),
    ],
  };

const characterPrototypesByAssetId: Record<string, EditorCharacterSpriteSheet> =
  {
    swordsman: swordsmanPrototype,
    knight: knightPrototype,
  };

function getCharacterDefaultSourceId(assetId: string) {
  return assetId.split("-copy-", 1)[0] ?? assetId;
}

function createFallbackCharacterPrototype(
  asset: Pick<ProjectAsset, "canvasSize">,
): EditorCharacterSpriteSheet {
  const dimensions = asset.canvasSize.match(/(\d+)\D+(\d+)/);
  const frameWidth = Number(dimensions?.[1]) || 64;
  const frameHeight = Number(dimensions?.[2]) || frameWidth;

  return {
    format: "png-sprite-sheet",
    imageUrl: "",
    frameWidth,
    frameHeight,
    columns: 1,
    rows: 1,
  };
}

function createFallbackCharacterAnimations(): EditorCharacterAnimation[] {
  return [{ id: "idle", label: "Idle", frameCount: 1 }];
}

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
    const sourceId = getCharacterDefaultSourceId(asset.id);
    return {
      mode: "character",
      ...base,
      character: {
        prototype: structuredClone(
          characterPrototypesByAssetId[sourceId] ??
            createFallbackCharacterPrototype(asset),
        ),
        animations: structuredClone(
          characterAnimationsByAssetId[sourceId] ??
            createFallbackCharacterAnimations(),
        ),
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
      prototype: saved.character.prototype ?? fallback.character.prototype,
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
