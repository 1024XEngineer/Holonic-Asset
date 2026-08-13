import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import {
  createAssetLibraryCacheSync,
  refreshAssetLibraryCache,
  removeAssetFromLibraryCache,
  restoreAssetLibraryCache,
} from "./asset-library-cache";
import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";

describe("createAssetLibraryCacheSync", () => {
  it("replaces only the targeted project library cache", () => {
    const queryClient = new QueryClient();
    const currentGroups: AssetGroup[] = [];
    const otherGroups: AssetGroup[] = [];
    const updatedGroups: AssetGroup[] = [{ kind: "object", assets: [] }];

    queryClient.setQueryData(assetKeys.library(7, "current"), currentGroups);
    queryClient.setQueryData(assetKeys.library(7, "other"), otherGroups);

    createAssetLibraryCacheSync(queryClient, 7)(updatedGroups, {
      projectId: "current",
    });

    expect(queryClient.getQueryData(assetKeys.library(7, "current"))).toEqual(
      updatedGroups,
    );
    expect(queryClient.getQueryData(assetKeys.library(7, "other"))).toEqual(
      otherGroups,
    );
  });

  it("refreshes a cached project library even while it is not mounted", async () => {
    const queryClient = new QueryClient();
    const key = assetKeys.library(7, "current");
    const currentGroups: AssetGroup[] = [];
    const updatedGroups: AssetGroup[] = [{ kind: "character", assets: [] }];
    const queryFn = vi.fn().mockResolvedValue(updatedGroups);
    queryClient.setQueryDefaults(key, { queryFn });
    queryClient.setQueryData(key, currentGroups);

    await refreshAssetLibraryCache(queryClient, 7, "current");

    expect(queryFn).toHaveBeenCalledOnce();
    expect(queryClient.getQueryData(key)).toEqual(updatedGroups);
  });
});

describe("asset library cache optimistic updates", () => {
  it("removes the targeted asset without waiting for a remote refresh", () => {
    const queryClient = new QueryClient();
    const key = assetKeys.library(7, "current");
    const groups: AssetGroup[] = [
      {
        kind: "object",
        assets: [
          { id: "8", name: "Barrel" },
          { id: "9", name: "Crate" },
        ] as AssetGroup["assets"],
      },
    ];
    queryClient.setQueryData(key, groups);

    const snapshot = removeAssetFromLibraryCache(
      queryClient,
      7,
      "current",
      "8",
    );

    expect(snapshot).toMatchObject({
      removed: { asset: { id: "8", name: "Barrel" }, assetIndex: 0 },
    });
    expect(queryClient.getQueryData(key)).toEqual([
      { kind: "object", assets: [{ id: "9", name: "Crate" }] },
    ]);
  });

  it("restores the prior data after a failed delete", () => {
    const queryClient = new QueryClient();
    const key = assetKeys.library(7, "current");
    const groups: AssetGroup[] = [
      {
        kind: "object",
        assets: [{ id: "8", name: "Barrel" }] as AssetGroup["assets"],
      },
    ];
    queryClient.setQueryData(key, groups);

    const snapshot = removeAssetFromLibraryCache(
      queryClient,
      7,
      "current",
      "8",
    );
    restoreAssetLibraryCache(queryClient, 7, "current", snapshot);

    expect(queryClient.getQueryData(key)).toEqual(groups);
  });

  it("does not change a loaded library when the asset is already absent", () => {
    const queryClient = new QueryClient();
    const key = assetKeys.library(7, "current");
    const groups: AssetGroup[] = [{ kind: "object", assets: [] }];
    queryClient.setQueryData(key, groups);

    const snapshot = removeAssetFromLibraryCache(
      queryClient,
      7,
      "current",
      "missing",
    );

    expect(snapshot).toEqual({});
    expect(queryClient.getQueryData(key)).toBe(groups);
  });

  it("rolls back only the failed asset during concurrent deletes", () => {
    const queryClient = new QueryClient();
    const key = assetKeys.library(7, "current");
    queryClient.setQueryData<AssetGroup[]>(key, [
      {
        kind: "object",
        assets: [
          { id: "8", name: "Barrel" },
          { id: "9", name: "Crate" },
        ] as AssetGroup["assets"],
      },
    ]);

    const first = removeAssetFromLibraryCache(queryClient, 7, "current", "8");
    removeAssetFromLibraryCache(queryClient, 7, "current", "9");
    restoreAssetLibraryCache(queryClient, 7, "current", first);

    expect(queryClient.getQueryData(key)).toEqual([
      { kind: "object", assets: [{ id: "8", name: "Barrel" }] },
    ]);
  });
});
