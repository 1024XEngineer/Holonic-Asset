import type { AssetGroup, AssetGroupsByProject } from "@/types/asset-library";
import type { ProjectAsset } from "@/types/asset";
import type { AssetRecord } from "@/types/record";
import type { ProjectSummary } from "@/types/project";

export const projectSummaries: ProjectSummary[] = [
  {
    id: "moonlit-orchard",
    name: "Moonlit Orchard",
    style: "Cozy pixel RPG",
    gameType: "Role-playing game",
    visualStyle: "Pixel art",
    platform: "PC",
    description:
      "A quiet orchard world where a young forager restores abandoned gardens by moonlight.",
    visualDirection: "",
    assetCount: 24,
  },
  {
    id: "iron-harbor",
    name: "Iron Harbor",
    style: "Industrial platformer",
    gameType: "Platformer",
    visualStyle: "Hand-painted",
    platform: "PC",
    description:
      "A mechanical harbor city built from cranes, furnaces, and weathered iron walkways.",
    visualDirection: "",
    assetCount: 18,
  },
  {
    id: "mushroom-courier",
    name: "Mushroom Courier",
    style: "Tiny adventure demo",
    gameType: "Role-playing game",
    visualStyle: "Cartoon",
    platform: "Web",
    description:
      "A cheerful mushroom courier delivers parcels through a miniature woodland kingdom.",
    visualDirection: "",
    assetCount: 31,
  },
];

function createHistory(
  assetId: string,
  currentVersion: string,
  currentDescription: string,
): AssetRecord[] {
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
    title: "Character",
    accentClassName: "bg-rose-500",
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
    title: "Object",
    accentClassName: "bg-amber-500",
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
    title: "Tiles",
    accentClassName: "bg-emerald-500",
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
    title: "Scenery",
    accentClassName: "bg-sky-500",
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
