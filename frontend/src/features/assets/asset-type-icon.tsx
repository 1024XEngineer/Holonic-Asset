import type { ComponentProps } from "react";

import { getAssetTypeConfig } from "./asset-type-config";
import type { CreatableAssetKind } from "./types";

export function AssetTypeIcon(
  _props: { kind: CreatableAssetKind } & ComponentProps<
    ReturnType<typeof getAssetTypeConfig>["icon"]
  >,
) {
  return null;
}
