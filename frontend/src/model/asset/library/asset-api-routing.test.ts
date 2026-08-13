import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  coreList: vi.fn(),
  coreDetail: vi.fn(),
  coreCopy: vi.fn(),
  coreDelete: vi.fn(),
  coreUpdate: vi.fn(),
}));

vi.mock("./core-asset.api", () => ({
  coreAssetApi: {
    list: mocks.coreList,
    detail: mocks.coreDetail,
    copy: mocks.coreCopy,
    delete: mocks.coreDelete,
    update: mocks.coreUpdate,
  },
}));
import { assetApi } from "./asset.api";

const remoteAssets = {
  assets: [
    {
      assetId: 8,
      projectId: 42,
      name: "Barrel",
      description: "Wooden prop",
      dimensions: { width: 32, height: 32 },
      perspective: "Top-Down",
      type: "object",
      version: 1,
      tags: ["prop"],
    },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.coreList.mockResolvedValue(remoteAssets);
  mocks.coreDetail.mockResolvedValue({
    assetId: 8,
    projectId: 42,
    name: "Barrel",
    description: "Wooden prop",
    dimensions: { width: 32, height: 32 },
    perspective: "Top-Down",
    type: "object",
    version: 1,
    tags: ["prop"],
    content: {
      prototype: [
        {
          id: 1,
          url: "https://storage.example/signed/barrel.png?token=abc",
        },
      ],
    },
  });
  mocks.coreCopy.mockResolvedValue({ newAssetId: 9 });
  mocks.coreDelete.mockResolvedValue({ assetId: 8 });
  mocks.coreUpdate.mockResolvedValue({});
});

describe("assetApi Core routing", () => {
  it("loads and maps assets from the Core API", async () => {
    await expect(assetApi.listGroups("42")).resolves.toEqual([
      expect.objectContaining({
        kind: "object",
        assets: [
          expect.objectContaining({
            id: "8",
            name: "Barrel",
            thumbnailUrl: "https://storage.example/signed/barrel.png?token=abc",
          }),
        ],
      }),
    ]);
    expect(mocks.coreList).toHaveBeenCalledWith(42);
    expect(mocks.coreDetail).toHaveBeenCalledWith(8);
  });

  it("keeps the library usable when an optional thumbnail detail fails", async () => {
    mocks.coreDetail.mockRejectedValueOnce(new Error("detail unavailable"));

    await expect(assetApi.listGroups("42")).resolves.toEqual([
      expect.objectContaining({
        kind: "object",
        assets: [expect.objectContaining({ id: "8" })],
      }),
    ]);
  });

  it("uses Core API copy and refreshes the remote library", async () => {
    await expect(assetApi.copy("42", "8")).resolves.toEqual([
      expect.objectContaining({ kind: "object" }),
    ]);

    expect(mocks.coreCopy).toHaveBeenCalledWith({ assetId: 8 });
    expect(mocks.coreList).toHaveBeenCalledWith(42);
  });

  it("uses Core API delete without blocking on a library refresh", async () => {
    await assetApi.delete("8");

    expect(mocks.coreDelete).toHaveBeenCalledWith({ assetId: 8 });
    expect(mocks.coreList).not.toHaveBeenCalled();
  });

  it("maps metadata updates to the Core API dimensions shape", async () => {
    await assetApi.update("42", "8", {
      name: "Large Barrel",
      description: "Updated prop",
      tags: ["prop", "wood"],
      canvasSize: "48 × 64 px",
      perspective: "Top-Down",
    });

    expect(mocks.coreUpdate).toHaveBeenCalledWith({
      assetId: 8,
      name: "Large Barrel",
      description: "Updated prop",
      tags: ["prop", "wood"],
      perspective: "Top-Down",
      dimensions: { width: 48, height: 64 },
    });
    expect(mocks.coreList).toHaveBeenCalledWith(42);
  });

  it("omits dimensions for non-dimensional assets", async () => {
    await assetApi.update("42", "8", {
      name: "Audio",
      description: "Updated audio",
      tags: [],
      canvasSize: "N/A",
      perspective: "Top-Down",
    });

    expect(mocks.coreUpdate).toHaveBeenCalledWith({
      assetId: 8,
      name: "Audio",
      description: "Updated audio",
      tags: [],
      perspective: "Top-Down",
    });
  });

  it("reports malformed dimensions as a domain error", async () => {
    await expect(
      assetApi.update("42", "8", {
        name: "Asset",
        description: "",
        tags: [],
        canvasSize: "not a size",
        perspective: "Top-Down",
      }),
    ).rejects.toThrow("Canvas size must use a positive width × height value.");
    expect(mocks.coreUpdate).not.toHaveBeenCalled();
  });

  it("rejects remote operations with non-persisted asset ids", async () => {
    await expect(assetApi.delete("barrel")).rejects.toThrow(
      "persisted Core API asset",
    );
    expect(mocks.coreDelete).not.toHaveBeenCalled();
  });
});
