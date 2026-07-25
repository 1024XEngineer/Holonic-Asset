/**
 * Custom Component: AssetTypeIcon
 * Renders the corresponding Lucide icon component dynamically based on the asset kind.
 */

import type { ComponentProps } from "react";
import type { CreatableAssetKind } from "@/domain/asset";
import { getAssetTypeConfig } from "./asset-type-config";

export function AssetTypeIcon({
  kind,
  ...props
}: { kind: CreatableAssetKind } & ComponentProps<
  ReturnType<typeof getAssetTypeConfig>["icon"]
>) {
  const Icon = getAssetTypeConfig(kind).icon;

  return <Icon {...props} />;
}
