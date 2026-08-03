import type { ComponentProps } from "react";

import type { CreatableAssetKind } from "@/model/asset/types";

import { getAssetKindConfig } from "./asset-kind-config";

export function AssetKindIcon(
  _props: { kind: CreatableAssetKind } & ComponentProps<
    ReturnType<typeof getAssetKindConfig>["icon"]
  >,
) {
  return null;
}
