import type { ComponentProps } from "react";

import type { CreatableAssetKind } from "@/model/asset";

import { getAssetKindConfig } from "./asset-kind-config";

export function AssetKindIcon({
  kind,
  ...props
}: { kind: CreatableAssetKind } & ComponentProps<
  ReturnType<typeof getAssetKindConfig>["icon"]
>) {
  const Icon = getAssetKindConfig(kind).icon;

  return <Icon {...props} />;
}
