import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import type { AssetLibraryItem } from "@/model/asset";
import { withI18n } from "@/testing/with-i18n";

const mocks = vi.hoisted(() => ({
  useRecordQuery: vi.fn(),
}));

vi.mock("@/model/asset", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/model/asset")>()),
  useRecordQuery: mocks.useRecordQuery,
}));

import { AssetPreview } from "./asset-preview";

describe("AssetPreview", () => {
  it("composites scenery layers when the list item has no thumbnail", () => {
    mocks.useRecordQuery.mockReturnValue({
      data: {
        record: {
          mode: "scenery",
          prompt: "Misty forest",
          scenery: {
            dimensions: { width: 160, height: 90 },
            layers: [
              {
                id: "sky",
                label: "Sky",
                detail: "Backdrop",
                imageUrl: "/sky.png",
                blendMode: "normal",
                position: { x: 0, y: 0 },
                transform: { scale: { x: 1, y: 1 }, rotation: 0 },
                opacity: 0.8,
                zIndex: 1,
              },
              {
                id: "trees",
                label: "Trees",
                detail: "Foreground",
                imageUrl: "/trees.png",
                blendMode: "multiply",
                visible: false,
              },
            ],
          },
        },
      },
      isPending: false,
      status: "success",
    });

    const html = renderToStaticMarkup(
      withI18n(<AssetPreview asset={sceneryAsset} projectId="42" />),
    );

    expect(mocks.useRecordQuery).toHaveBeenCalledWith("42", "8");
    expect(html).toContain('aria-label="Forest preview"');
    expect(html).toContain('src="/sky.png"');
    expect(html).not.toContain('src="/trees.png"');
    expect(html).toContain("aspect-ratio:160 / 90");
    expect(html).toContain("opacity:0.8");
  });
});

const sceneryAsset: AssetLibraryItem = {
  id: "8",
  kind: "scenery",
  name: "Forest",
  description: "Misty forest",
  version: "v1",
  canvasSize: "160 × 90 px",
  perspective: "Top-Down",
  tags: [],
  history: [],
  animations: [],
};
