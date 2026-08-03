import type { ComponentProps } from "react";

import type { CreatableAssetKind } from "@/model/asset";

import { getAssetKindConfig } from "./asset-kind-config";

export function AssetKindIcon(
  _props: { kind: CreatableAssetKind } & ComponentProps<
    ReturnType<typeof getAssetKindConfig>["icon"]
  >,
) {
  return null;
}
