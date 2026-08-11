import type {
  AssetKind,
  CharacterAnimation,
  CharacterAnimationClip,
  CharacterSpriteSheet,
  ProjectAsset,
} from "../../types";
import type {
  AssetRecord,
  CharacterAssetRecord,
  TilesetItem,
  AssetRecordForKind,
  SceneryAssetRecord,
  TilesetAssetRecord,
  UISetAssetRecord,
  AudioAssetRecord,
} from "../types";
import type {
  ObjectAssetRecord,
  SpriteAssetRecordData,
} from "../types/asset-record";

const swordsmanPrototype: CharacterSpriteSheet = {
  format: "png-sprite-sheet",
  imageUrl: "/assets/characters/swordsman/prototype.png",
  frameWidth: 64,
  frameHeight: 64,
  columns: 4,
  rows: 1,
};

const knightPrototype: CharacterSpriteSheet = {
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
): CharacterAnimationClip {
  return {
    kind: "clip",
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

function createSwordsmanAnimations(
  id: string,
  label: string,
  frameCounts: Record<(typeof swordsmanDirections)[number]["id"], number>,
): CharacterAnimation[] {
  return swordsmanDirections.map((direction) =>
    createPngAnimation(
      `${id}-${direction.id}`,
      `${label} ${direction.label}`,
      `/assets/characters/swordsman/${id}-${direction.id}.png`,
      frameCounts[direction.id],
      64,
      64,
    ),
  );
}

const spriteAnimationsByAssetId: Record<string, CharacterAnimation[]> = {
  swordsman: [
    ...createSwordsmanAnimations("idle", "Idle", {
      front: 12,
      back: 4,
      left: 12,
      right: 12,
    }),
    ...createSwordsmanAnimations("attack", "Attack", {
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
  "alchemy-table": [
    createPngAnimation(
      "idle",
      "Idle",
      "/assets/object/Alchemy_Table_02-Sheet.png",
      8,
      48,
      64,
    ),
  ],
};

const spritePrototypesByAssetId: Record<string, CharacterSpriteSheet> = {
  swordsman: swordsmanPrototype,
  knight: knightPrototype,
  "alchemy-table": {
    format: "png-sprite-sheet",
    imageUrl: "/assets/object/Alchemy_Table_02-Sheet.png",
    frameWidth: 48,
    frameHeight: 64,
    columns: 1,
    rows: 1,
  },
};

function getSpriteDefaultSourceId(assetId: string) {
  return assetId.split("-copy-", 1)[0] ?? assetId;
}

function createFallbackSpritePrototype(
  asset: Pick<ProjectAsset, "canvasSize" | "thumbnailUrl">,
): CharacterSpriteSheet {
  const dimensions = asset.canvasSize.match(/(\d+)\D+(\d+)/);
  const frameWidth = Number(dimensions?.[1]) || 64;
  const frameHeight = Number(dimensions?.[2]) || frameWidth;

  return {
    format: "png-sprite-sheet",
    imageUrl: asset.thumbnailUrl ?? "",
    frameWidth,
    frameHeight,
    columns: 1,
    rows: 1,
  };
}

function createFallbackSpriteAnimations(): CharacterAnimation[] {
  return [{ kind: "clip", id: "idle", label: "Idle", frameCount: 1 }];
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
  tiles: TilesetItem["tiles"];
}): TilesetItem {
  return {
    id,
    label,
    imageUrl: `${tilesetAssetBase}/${imageUrl}`,
    tiles,
  };
}

const tilesetItems: TilesetItem[] = [
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

export function createDefaultAssetRecord<K extends AssetKind>(
  kind: K,
  asset: ProjectAsset,
): AssetRecordForKind<K> {
  const base = { prompt: asset.description };

  if (kind === "character") {
    return {
      mode: "character",
      ...base,
      character: createDefaultSpriteAssetRecordData(asset),
    } as AssetRecordForKind<K>;
  }

  if (kind === "object") {
    return {
      mode: "object",
      ...base,
      object: createDefaultSpriteAssetRecordData(asset),
    } as AssetRecordForKind<K>;
  }

  if (kind === "scenery") {
    return {
      mode: "scenery",
      ...base,
      scenery: {
        layers: structuredClone(asset.scenery?.layers ?? []),
      },
    } as AssetRecordForKind<K>;
  }

  if (kind === "tileset") {
    return {
      mode: "tileset",
      ...base,
      tileset: {
        gridSize: 8,
        items: structuredClone(tilesetItems),
      },
    } as AssetRecordForKind<K>;
  }

  if (kind === "uiset") {
    return {
      mode: "uiset",
      ...base,
      uiset: {
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
    } as AssetRecordForKind<K>;
  }

  return {
    mode: "audio",
    ...base,
    audio: {},
  } as AssetRecordForKind<K>;
}

export function mergeAssetRecord<K extends AssetKind>(
  kind: K,
  fallback: AssetRecordForKind<K>,
  saved: AssetRecord | LegacySpriteSheetRecord | undefined,
): AssetRecordForKind<K> {
  const migrated = migrateLegacyTilesetRecord(kind, saved);
  const record = migrated ?? saved;

  if (!record) {
    return fallback;
  }

  switch (fallback.mode) {
    case "character":
      return mergeCharacterRecord(
        fallback as CharacterAssetRecord,
        record as CharacterAssetRecord,
      ) as AssetRecordForKind<K>;
    case "object":
      return mergeObjectRecord(
        fallback as ObjectAssetRecord,
        record as ObjectAssetRecord,
      ) as AssetRecordForKind<K>;
    case "scenery":
      return mergeSceneryRecord(
        record as SceneryAssetRecord,
      ) as AssetRecordForKind<K>;
    case "tileset":
      return mergeTilesetRecord(
        record as TilesetAssetRecord,
      ) as AssetRecordForKind<K>;
    case "uiset":
      return mergeUISetRecord(
        record as UISetAssetRecord,
      ) as AssetRecordForKind<K>;
    case "audio":
      return mergeAudioRecord(
        record as AudioAssetRecord,
      ) as AssetRecordForKind<K>;
  }
}

function createDefaultSpriteAssetRecordData(
  asset: ProjectAsset,
): SpriteAssetRecordData {
  const sourceId = getSpriteDefaultSourceId(asset.id);

  return {
    prototype: structuredClone(
      spritePrototypesByAssetId[sourceId] ??
        createFallbackSpritePrototype(asset),
    ),
    animations: structuredClone(
      spriteAnimationsByAssetId[sourceId] ?? createFallbackSpriteAnimations(),
    ),
    nodePositions: {},
  };
}

function migrateLegacyTilesetRecord(
  kind: AssetKind,
  saved: AssetRecord | LegacySpriteSheetRecord | undefined,
): TilesetAssetRecord | undefined {
  if (kind !== "tileset" || saved?.mode !== "sprite-sheet") {
    return undefined;
  }

  return {
    mode: "tileset",
    prompt: saved.prompt,
    tileset: saved.spriteSheet,
  };
}

type LegacySpriteSheetRecord = {
  mode: "sprite-sheet";
  prompt: string;
  spriteSheet: TilesetAssetRecord["tileset"];
};

function mergeCharacterRecord(
  fallback: CharacterAssetRecord,
  saved: CharacterAssetRecord,
): CharacterAssetRecord {
  return {
    mode: "character",
    prompt: saved.prompt,
    character: mergeSpriteAssetRecordData(fallback.character, saved.character),
  };
}

function mergeObjectRecord(
  fallback: ObjectAssetRecord,
  saved: ObjectAssetRecord,
): ObjectAssetRecord {
  return {
    mode: "object",
    prompt: saved.prompt,
    object: mergeSpriteAssetRecordData(fallback.object, saved.object),
  };
}

function mergeSpriteAssetRecordData(
  fallback: SpriteAssetRecordData,
  saved: SpriteAssetRecordData,
): SpriteAssetRecordData {
  return {
    prototype: saved.prototype ?? fallback.prototype,
    animations: saved.animations ?? fallback.animations ?? [],
    nodePositions: {
      ...fallback.nodePositions,
      ...saved.nodePositions,
    },
  };
}

function mergeSceneryRecord(saved: SceneryAssetRecord): SceneryAssetRecord {
  return {
    mode: "scenery",
    prompt: saved.prompt,
    scenery: { layers: saved.scenery.layers },
  };
}

function mergeTilesetRecord(saved: TilesetAssetRecord): TilesetAssetRecord {
  return {
    mode: "tileset",
    prompt: saved.prompt,
    tileset: {
      gridSize: saved.tileset.gridSize,
      items: saved.tileset.items,
    },
  };
}

function mergeUISetRecord(saved: UISetAssetRecord): UISetAssetRecord {
  return {
    mode: "uiset",
    prompt: saved.prompt,
    uiset: { components: saved.uiset.components },
  };
}

function mergeAudioRecord(saved: AudioAssetRecord): AudioAssetRecord {
  return { mode: "audio", prompt: saved.prompt, audio: {} };
}
