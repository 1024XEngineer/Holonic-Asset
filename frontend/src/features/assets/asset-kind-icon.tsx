import type { ComponentProps } from "react";

import { getAssetKindConfig } from "./asset-kind-config";
import type { CreatableAssetKind } from "./types";

export function AssetKindIcon(
  _props: { kind: CreatableAssetKind } & ComponentProps<
    ReturnType<typeof getAssetKindConfig>["icon"]
  >,
) {
  return null;
}
