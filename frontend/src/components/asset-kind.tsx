import type { ComponentProps } from "react";
import type { LucideIcon } from "lucide-react";
import {
  Box,
  Grid3X3,
  Mountain,
  PanelsTopLeft,
  UserRound,
  Volume2,
} from "lucide-react";

import type { AssetKind } from "@/model/asset";

export type AssetKindConfig = {
  icon: LucideIcon;
  accentClassName: string;
};

const assetKindConfigs: Record<AssetKind, AssetKindConfig> = {
  character: {
    icon: UserRound,
    accentClassName: "bg-rose-500",
  },
  object: { icon: Box, accentClassName: "bg-amber-500" },
  tileset: {
    icon: Grid3X3,
    accentClassName: "bg-emerald-500",
  },
  scenery: { icon: Mountain, accentClassName: "bg-sky-500" },
  uiset: {
    icon: PanelsTopLeft,
    accentClassName: "bg-slate-500",
  },
  audio: { icon: Volume2, accentClassName: "bg-slate-500" },
};

export function getAssetKindConfig(kind: AssetKind) {
  return assetKindConfigs[kind];
}

export function AssetKindIcon({
  kind,
  ...props
}: { kind: AssetKind } & ComponentProps<
  ReturnType<typeof getAssetKindConfig>["icon"]
>) {
  const Icon = getAssetKindConfig(kind).icon;

  return <Icon {...props} />;
}
