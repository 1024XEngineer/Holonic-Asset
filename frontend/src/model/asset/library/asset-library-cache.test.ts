import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

import {
  createAssetLibraryCacheSync,
  refreshAssetLibraryCache,
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
