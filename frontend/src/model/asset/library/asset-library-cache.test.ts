import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { createAssetLibraryCacheSync } from "./asset-library-cache";
import { assetKeys } from "./keys";
import type { AssetGroup } from "./types";

describe("createAssetLibraryCacheSync", () => {
  it("replaces only the targeted project library cache", () => {
    const queryClient = new QueryClient();
    const currentGroups: AssetGroup[] = [];
    const otherGroups: AssetGroup[] = [];
    const updatedGroups: AssetGroup[] = [{ kind: "object", assets: [] }];

    queryClient.setQueryData(assetKeys.library("current"), currentGroups);
    queryClient.setQueryData(assetKeys.library("other"), otherGroups);

    createAssetLibraryCacheSync(queryClient)(updatedGroups, {
      projectId: "current",
    });

    expect(queryClient.getQueryData(assetKeys.library("current"))).toEqual(
      updatedGroups,
    );
    expect(queryClient.getQueryData(assetKeys.library("other"))).toEqual(
      otherGroups,
    );
  });
});
