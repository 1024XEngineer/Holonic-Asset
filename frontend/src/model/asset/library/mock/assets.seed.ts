import type { AssetRevision, ProjectAsset } from "../../types";
import type { AssetGroup, AssetGroupsByProject } from "../types";

function createHistory(
  assetId: string,
  currentVersion: string,
  currentDescription: string,
): AssetRevision[] {
  const currentNumber = Number.parseInt(currentVersion.replace("v", ""), 10);
  const descriptions = [
    currentDescription,
    "Adjusted contrast and edge cleanup",
    "Initial generated concept",
  ];

  return Array.from(
    { length: Math.min(3, Math.max(1, currentNumber)) },
    (_, index) => {
      const version = `v${currentNumber - index}`;

      return {
        id: `${assetId}-history-${version}`,
        version,
        description: descriptions[index],
        status: "ready" as const,
        isCurrent: index === 0,
      };
    },
  );
}

function createAsset(
  asset: Omit<ProjectAsset, "history" | "animations">,
): ProjectAsset {
  return {
    ...asset,
    history: createHistory(asset.id, asset.version, asset.description),
    animations: [],
  };
}

const moonlitOrchardAssetGroups: AssetGroup[] = [
  {
    kind: "character",
    assets: [
      createAsset({
        id: "swordsman",
        name: "Swordsman",
        description: "Four-direction top-down swordsman",
        version: "v1",
        canvasSize: "64 × 64 px",
        perspective: "Top-down",
        tags: ["swordsman", "four-direction", "pixel-art"],
        thumbnailUrl: "/assets/characters/swordsman/prototype.png",
        previewFrame: { columns: 4, rows: 1, column: 0, row: 0 },
      }),
      createAsset({
        id: "knight",
        name: "Knight",
        description: "Single-direction knight sprite",
        version: "v1",
        canvasSize: "128 × 128 px",
        perspective: "Side-On",
        tags: ["knight", "single-direction", "pixel-art"],
        thumbnailUrl: "/assets/characters/knight/prototype.png",
        previewOffset: { x: "8%", y: "-10%" },
      }),
    ],
  },
  {
    kind: "object",
    assets: [
      createAsset({
        id: "alchemy-table",
        name: "Alchemy Table",
        description: "48x64 alchemy workstation sprite",
        version: "v3",
        canvasSize: "48 × 64 px",
        perspective: "Top-down",
        tags: ["alchemy", "table", "workstation", "pixel-art"],
        thumbnailUrl: "/assets/object/Alchemy_Table_02-Sheet.png",
        previewCrop: {
          sourceWidth: 528,
          sourceHeight: 320,
          x: 0,
          y: 0,
          width: 48,
          height: 64,
          displayOffsetY: "-6%",
        },
      }),
    ],
  },
  {
    kind: "tileset",
    assets: [
      createAsset({
        id: "orchard-ground-set",
        name: "Orchard Ground Set",
        description: "Grass, dirt, path edges",
        version: "v7",
        canvasSize: "16 × 16 px",
        perspective: "Top-down",
        tags: ["terrain", "ground"],
      }),
    ],
  },
  {
    kind: "scenery",
    assets: [
      createAsset({
        id: "moonlit-orchard-scene",
        name: "Moonlit Orchard Scene",
        description: "Sky, hills, trees, and foreground layers",
        version: "v3",
        canvasSize: "1920 × 1080 px",
        perspective: "Side-On",
        tags: ["environment", "orchard"],
        thumbnailUrl: "/assets/nearby-trees-clean.png",
        previewOffset: { x: "0", y: "20%" },
        previewScale: 1.15,
        scenery: {
          layers: [
            {
              id: "sky",
              label: "Sky",
              detail: "Background layer",
              imageUrl: "/assets/sky.png",
              blendMode: "normal",
            },
            {
              id: "wind",
              label: "Wind",
              detail: "Atmosphere layer",
              imageUrl: "/assets/wind.png",
              blendMode: "multiply",
            },
            {
              id: "nearby-trees",
              label: "Nearby trees",
              detail: "Foreground layer",
              imageUrl: "/assets/nearby-trees.png",
              blendMode: "multiply",
            },
          ],
        },
      }),
    ],
  },
  {
    kind: "ui",
    assets: [
      createAsset({
        id: "quest-log-ui",
        name: "Quest Log",
        description: "Parchment quest tracker with a primary action",
        version: "v1",
        canvasSize: "320 × 180 px",
        perspective: "Top-down",
        tags: ["interface", "quest", "parchment"],
        thumbnailUrl: "/assets/ui/ui.png",
      }),
    ],
  },
];

export const assetGroupsByProject: AssetGroupsByProject = {
  "moonlit-orchard": moonlitOrchardAssetGroups,
  "iron-harbor": [],
  "mushroom-courier": [],
};
