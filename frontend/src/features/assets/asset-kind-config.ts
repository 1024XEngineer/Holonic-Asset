import type { LucideIcon } from "lucide-react";
import {
  Box,
  Grid3X3,
  Mountain,
  PanelsTopLeft,
  UserRound,
  Volume2,
} from "lucide-react";

import type { AssetKind } from "./types";

export type AssetKindConfig = {
  label: string;
  icon: LucideIcon;
  accentClassName: string;
};

const assetKindConfigs: Record<AssetKind, AssetKindConfig> = {
  character: {
    label: "Character",
    icon: UserRound,
    accentClassName: "bg-rose-500",
  },
  object: { label: "Object", icon: Box, accentClassName: "bg-amber-500" },
  tileset: {
    label: "Tileset",
    icon: Grid3X3,
    accentClassName: "bg-emerald-500",
  },
  scenery: { label: "Scenery", icon: Mountain, accentClassName: "bg-sky-500" },
  ui: { label: "UI", icon: PanelsTopLeft, accentClassName: "bg-slate-500" },
  audio: { label: "Audio", icon: Volume2, accentClassName: "bg-slate-500" },
};

export function getAssetKindConfig(kind: AssetKind) {
  return assetKindConfigs[kind];
}
