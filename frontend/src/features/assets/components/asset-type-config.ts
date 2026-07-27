/**
 * Custom Config Module: asset-type-config
 * Configures UI labels, Lucide icons, and preview colors for each asset kind.
 */

import type { LucideIcon } from "lucide-react";
import {
  Box,
  Grid3X3,
  Image,
  Mountain,
  PanelsTopLeft,
  UserRound,
  Volume2,
} from "lucide-react";

import type { AssetKind } from "@/features/assets/domain";

export type AssetTypeConfig = {
  label: string;
  icon: LucideIcon;
  accentClassName: string;
};

const assetTypeConfigs: Record<AssetKind, AssetTypeConfig> = {
  character: {
    label: "Character",
    icon: UserRound,
    accentClassName: "bg-rose-500",
  },
  object: { label: "Object", icon: Box, accentClassName: "bg-amber-500" },
  tiles: { label: "Tiles", icon: Grid3X3, accentClassName: "bg-emerald-500" },
  scenery: { label: "Scenery", icon: Mountain, accentClassName: "bg-sky-500" },
  audio: { label: "Audio", icon: Volume2, accentClassName: "bg-slate-500" },
  background: {
    label: "Background",
    icon: Image,
    accentClassName: "bg-slate-500",
  },
  ui: { label: "UI", icon: PanelsTopLeft, accentClassName: "bg-slate-500" },
};

export function getAssetTypeConfig(kind: AssetKind) {
  return assetTypeConfigs[kind];
}
