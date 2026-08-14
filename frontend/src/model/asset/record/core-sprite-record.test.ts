import { describe, expect, it } from "vitest";

import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import { toCoreSpriteAssetWorkspace } from "./core-sprite-record";

describe("toCoreSpriteAssetWorkspace", () => {
  it("maps Core animation frames into an editor sprite record", () => {
    const workspace = toCoreSpriteAssetWorkspace({
      projectId: "11",
      projectName: "Demo",
      detail: characterDetail(),
      records: [
        {
          recordId: 31,
          assetId: 9,
          version: 3,
          contentId: 41,
          createdAt: "2026-08-14T01:00:00Z",
          name: "Hero",
          description: "A generated hero",
          perspective: "Top-Down",
          dimensions: { width: 32, height: 32 },
        },
      ] satisfies AssetRecordResponse<"character">[],
    });

    expect(workspace).toMatchObject({
      projectName: "Demo",
      asset: {
        id: "9",
        projectId: "11",
        kind: "character",
        version: "v3",
        history: [
          {
            id: "31",
            version: "v3",
            status: "ready",
            isCurrent: true,
          },
        ],
      },
      record: {
        mode: "character",
        prompt: "A generated hero",
        character: {
          prototype: {
            imageUrl: "/front.png",
            frameUrls: ["/front.png", "/right.png", "/back.png", "/left.png"],
            frameWidth: 32,
            frameHeight: 32,
            columns: 2,
            rows: 2,
          },
          animations: [
            {
              kind: "clip",
              id: "7",
              label: "Walk",
              frameCount: 2,
              spriteSheet: {
                imageUrl: "/walk-1.png",
                frameUrls: ["/walk-1.png", "/walk-2.png"],
                frameWidth: 32,
                frameHeight: 32,
                columns: 2,
                rows: 1,
              },
            },
          ],
          nodePositions: {},
        },
      },
    });
  });
});

function characterDetail(): AssetDetailResponse {
  return {
    assetId: 9,
    projectId: 11,
    name: "Hero",
    description: "A generated hero",
    type: "character",
    perspective: "Top-Down",
    dimensions: { width: 32, height: 32 },
    tags: [],
    version: 3,
    content: {
      directionCount: 4,
      prototype: [
        { id: 1, url: "/front.png" },
        { id: 2, url: "/right.png" },
        { id: 3, url: "/back.png" },
        { id: 4, url: "/left.png" },
      ],
      animations: [
        {
          id: 7,
          name: "Walk",
          frames: [
            { id: 1, url: "/walk-1.png", duration: 83 },
            { id: 2, url: "/walk-2.png", duration: 83 },
          ],
        },
      ],
    },
  };
}
