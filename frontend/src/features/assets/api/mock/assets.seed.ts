import type {
  AssetGroup,
  AssetGroupsByProject,
} from "@/features/assets/domain";
import type { ProjectAsset } from "@/features/assets/domain";
import type { AssetRevision } from "@/features/assets/domain";

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
        id: "forager-hero",
        name: "Forager Hero",
        description: "Idle, walk, harvest",
        version: "v4",
        canvasSize: "32 × 32 px",
        perspective: "Top-down",
        tags: ["hero", "orchard"],
      }),
      createAsset({
        id: "lantern-merchant",
        name: "Lantern Merchant",
        description: "Front view draft",
        version: "v2",
        canvasSize: "32 × 32 px",
        perspective: "Front view",
        tags: ["npc", "merchant"],
      }),
      createAsset({
        id: "moss-slime",
        name: "Moss Slime",
        description: "Bounce animation",
        version: "v6",
        canvasSize: "32 × 32 px",
        perspective: "Top-down",
        tags: ["creature", "enemy"],
      }),
    ],
  },
  {
    kind: "object",
    assets: [
      createAsset({
        id: "copper-watering-can",
        name: "Copper Watering Can",
        description: "32x32 item sprite",
        version: "v3",
        canvasSize: "32 × 32 px",
        perspective: "Top-down",
        tags: ["tool", "copper"],
      }),
      createAsset({
        id: "blueberry-crate",
        name: "Blueberry Crate",
        description: "4 color variants",
        version: "v1",
        canvasSize: "32 × 32 px",
        perspective: "Three-quarter",
        tags: ["prop", "storage"],
      }),
      createAsset({
        id: "weathered-signpost",
        name: "Weathered Signpost",
        description: "Directional prop",
        version: "v5",
        canvasSize: "32 × 32 px",
        perspective: "Front view",
        tags: ["prop", "wood"],
      }),
    ],
  },
  {
    kind: "tiles",
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
      createAsset({
        id: "stone-wall-corners",
        name: "Stone Wall Corners",
        description: "Autotile pieces",
        version: "v2",
        canvasSize: "16 × 16 px",
        perspective: "Top-down",
        tags: ["terrain", "wall"],
      }),
      createAsset({
        id: "pond-rim-tiles",
        name: "Pond Rim Tiles",
        description: "Water border kit",
        version: "v3",
        canvasSize: "16 × 16 px",
        perspective: "Top-down",
        tags: ["terrain", "water"],
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
        perspective: "Side view",
        tags: ["environment", "orchard"],
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
];

export const assetGroupsByProject: AssetGroupsByProject = {
  "moonlit-orchard": moonlitOrchardAssetGroups,
  "iron-harbor": [],
  "mushroom-courier": [],
};
