import type { Perspective } from "@/model/project";
import { z } from "zod";

export const assetDirections = [
  "front",
  "front_right",
  "right",
  "back_right",
  "back",
  "back_left",
  "left",
  "front_left",
] as const;

export const assetDirectionSchema = z.enum(assetDirections);

export type AssetDirection = z.infer<typeof assetDirectionSchema>;

export const assetDirectionsByPerspective = {
  "Side-On": ["left", "right"],
  "Top-Down": ["front", "right", "back", "left"],
  Isometric: assetDirections,
} as const satisfies Record<Perspective, readonly AssetDirection[]>;
