import type { AssetKind } from "../types";
import type { EditorRecord } from "./types";

export function editorModeForAssetKind(kind: AssetKind): EditorRecord["mode"] {
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
