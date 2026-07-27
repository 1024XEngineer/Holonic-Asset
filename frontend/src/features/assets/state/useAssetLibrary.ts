import { useMemo, useState } from "react";

import { getAssetTypeConfig } from "../components/asset-type-config";
import type { AssetGroup } from "@/features/assets/domain";
import { assetKinds, type AssetKind } from "@/features/assets/domain";

export function useAssetLibrary(assetGroups: AssetGroup[], query: string) {
  const [selectedKinds, setSelectedKinds] = useState<AssetKind[]>([
    ...assetKinds,
  ]);
  const filteredAssets = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();

    return assetGroups
      .filter((group) => selectedKinds.includes(group.kind))
      .flatMap((group) =>
        group.assets
          .filter((asset) =>
            [asset.name, asset.description, asset.version].some((value) =>
              value.toLowerCase().includes(normalizedQuery),
            ),
          )
          .map((asset) => ({
            ...asset,
            kind: group.kind,
            accentClassName: getAssetTypeConfig(group.kind).accentClassName,
            kindLabel: getAssetTypeConfig(group.kind).label,
          })),
      );
  }, [assetGroups, query, selectedKinds]);

  return { filteredAssets, selectedKinds, setSelectedKinds };
}
