import type { AssetKind } from "../types";
import type { AssetRecord } from "./types";

export function assetRecordModeForKind(kind: AssetKind): AssetRecord["mode"] {
  switch (kind) {
    case "character":
    case "object":
      return "character";
    case "scenery":
      return "scenery";
    case "tileset":
      return "tileset";
    case "ui":
      return "ui";
    case "audio":
      return "audio";
  }
}
