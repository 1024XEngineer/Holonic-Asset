import type { AssetKind, ProjectAsset } from "@/features/assets/domain";
import type {
  CharacterEditorDocument,
  EditorCharacterAnimation,
  EditorCharacterAnimationClip,
  EditorCharacterAnimationGroup,
  EditorCharacterSpriteSheet,
  EditorTilesetItem,
  EditorDocumentForKind,
  SceneryEditorDocument,
  TilesetEditorDocument,
  UiEditorDocument,
  AudioEditorDocument,
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
  tiles: EditorTilesetItem["tiles"];
}): EditorTilesetItem {
  return {
    id,
    label,
    imageUrl: `${tilesetAssetBase}/${imageUrl}`,
    tiles,
  };
}

const tilesetItems: EditorTilesetItem[] = [
  createTilesetItem({
    id: "sofa-01",
    label: "Sofa 01",
    imageUrl: "sofas/sofa_01_96x64.png",
    tiles: [
      [0, 0],
      [1, 0],
      [2, 0],
      [0, 1],
      [1, 1],
      [2, 1],
    ],
  }),
  createTilesetItem({
    id: "sofa-02",
    label: "Sofa 02",
    imageUrl: "sofas/sofa_02_96x64.png",
    tiles: [
      [3, 0],
      [4, 0],
      [5, 0],
      [3, 1],
      [4, 1],
      [5, 1],
    ],
  }),
  createTilesetItem({
    id: "sofa-03",
    label: "Sofa 03",
    imageUrl: "sofas/sofa_03_96x64.png",
    tiles: [
      [0, 2],
      [1, 2],
      [2, 2],
      [0, 3],
      [1, 3],
      [2, 3],
    ],
  }),
  createTilesetItem({
    id: "sofa-04",
    label: "Sofa 04",
    imageUrl: "sofas/sofa_04_96x64.png",
    tiles: [
      [3, 2],
      [4, 2],
      [5, 2],
      [3, 3],
      [4, 3],
      [5, 3],
    ],
  }),
  createTilesetItem({
    id: "sofa-05",
    label: "Sofa 05",
    imageUrl: "sofas/sofa_05_96x64.png",
    tiles: [
      [0, 4],
      [1, 4],
      [2, 4],
      [0, 5],
      [1, 5],
      [2, 5],
    ],
  }),
  ...Array.from({ length: 4 }, (_, index) =>
    createTilesetItem({
      id: `bed-${index + 1}`,
      label: `Bed ${String(index + 1).padStart(2, "0")}`,
      imageUrl: `beds/bed_${String(index + 1).padStart(2, "0")}_32x64.png`,
      tiles: [
        [6, index * 2],
        [6, index * 2 + 1],
      ],
    }),
  ),
  ...[
    "barrel_01",
    "barrel_02",
    "barrel_03",
    "barrel_04",
    "barrel_05",
    "jar_01",
    "jar_02",
    "jar_03",
  ].map((name, index) =>
    createTilesetItem({
      id: name.replace("_", "-"),
      label: name
        .replace("_", " ")
        .replace(/\b\w/g, (letter) => letter.toUpperCase()),
      imageUrl: `items/${name}_32x32.png`,
      tiles: [[7, index]],
    }),
  ),
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

  if (kind === "tileset") {
    return {
      mode: "tileset",
      ...base,
      tileset: {
        gridSize: 8,
        items: structuredClone(tilesetItems),
      },
    } as EditorDocumentForKind<K>;
  }

  if (kind === "ui") {
    return {
      mode: "ui",
      ...base,
      ui: {
        components: [
          {
            id: "panel",
            label: "Panel",
            kind: "panel",
            bounds: { x: 8, y: 12, width: 84, height: 76 },
          },
          {
            id: "title",
            label: asset.name,
            kind: "label",
            bounds: { x: 16, y: 23, width: 68, height: 12 },
          },
          {
            id: "primary-action",
            label: "Primary action",
            kind: "button",
            bounds: { x: 28, y: 63, width: 44, height: 14 },
          },
        ],
      },
    } as EditorDocumentForKind<K>;
  }

  return {
    mode: "audio",
    ...base,
    audio: {},
  } as EditorDocumentForKind<K>;
}

export function mergeEditorDocument<K extends AssetKind>(
  kind: K,
  fallback: EditorDocumentForKind<K>,
  saved: unknown,
): EditorDocumentForKind<K> {
  const migrated = migrateLegacyTilesetDocument(kind, saved);
  const document = migrated ?? saved;

  if (!document || !isEditorDocumentForAssetKind(kind, document)) {
    return fallback;
  }

  switch (fallback.mode) {
    case "character":
      return mergeCharacterRecord(
        fallback as CharacterEditorDocument,
        document as CharacterEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "scenery":
      return mergeSceneryRecord(
        document as SceneryEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "tileset":
      return mergeTilesetRecord(
        document as TilesetEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "ui":
      return mergeUiRecord(
        document as UiEditorDocument,
      ) as EditorDocumentForKind<K>;
    case "audio":
      return mergeAudioRecord(
        document as AudioEditorDocument,
      ) as EditorDocumentForKind<K>;
  }
}

function migrateLegacyTilesetDocument(
  kind: AssetKind,
  saved: unknown,
): TilesetEditorDocument | undefined {
  if (kind !== "tileset" || !isLegacySpriteSheetDocument(saved)) {
    return undefined;
  }

  const migrated: TilesetEditorDocument = {
    mode: "tileset",
    prompt: saved.prompt,
    tileset: saved.spriteSheet,
  };

  return isEditorDocumentForAssetKind("tileset", migrated)
    ? migrated
    : undefined;
}

function isLegacySpriteSheetDocument(value: unknown): value is {
  mode: "sprite-sheet";
  prompt: string;
  spriteSheet: TilesetEditorDocument["tileset"];
} {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return false;
  }

  const document = value as Record<string, unknown>;
  return (
    document.mode === "sprite-sheet" &&
    typeof document.prompt === "string" &&
    typeof document.spriteSheet === "object" &&
    document.spriteSheet !== null &&
    !Array.isArray(document.spriteSheet)
  );
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

function mergeTilesetRecord(
  saved: TilesetEditorDocument,
): TilesetEditorDocument {
  return {
    mode: "tileset",
    prompt: saved.prompt,
    tileset: {
      gridSize: saved.tileset.gridSize,
      items: saved.tileset.items,
    },
  };
}

function mergeUiRecord(saved: UiEditorDocument): UiEditorDocument {
  return {
    mode: "ui",
    prompt: saved.prompt,
    ui: { components: saved.ui.components },
  };
}

function mergeAudioRecord(saved: AudioEditorDocument): AudioEditorDocument {
  return { mode: "audio", prompt: saved.prompt, audio: {} };
}
