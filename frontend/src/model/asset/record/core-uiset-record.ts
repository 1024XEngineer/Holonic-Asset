import type {
  AssetDetailResponse,
  AssetRecordResponse,
} from "../library/asset.contract";
import type { AssetRecord, AssetWorkspaceData, UISetComponent } from "./types";
import { createCoreAssetWorkspace } from "./core-asset-workspace";

export function toCoreUISetAssetWorkspace({
  projectId,
  projectName,
  detail,
  records,
}: {
  projectId: string;
  projectName: string;
  detail: Extract<AssetDetailResponse, { type: "uiset" }>;
  records: AssetRecordResponse[];
}): AssetWorkspaceData {
  const { width, height } = detail.dimensions;
  const record: AssetRecord = {
    mode: "uiset",
    prompt: detail.description,
    uiset: {
      dimensions: { width, height },
      components: (detail.content?.components ?? []).map((component) => ({
        id: String(component.id),
        label: component.name,
        kind: readUISetComponentKind(component.metadata),
        bounds: {
          x: toPercentage(component.position.x, width),
          y: toPercentage(component.position.y, height),
          width: toPercentage(component.size.width, width),
          height: toPercentage(component.size.height, height),
        },
      })),
    },
  };

  return createCoreAssetWorkspace({
    projectId,
    projectName,
    detail,
    kind: "uiset",
    record,
    records,
  });
}

export function toCoreUISetAssetContent(
  record: Extract<AssetRecord, { mode: "uiset" }>,
) {
  return {
    components: record.uiset.components.map((component, componentIndex) =>
      toCoreUISetComponent(component, componentIndex, record.uiset.dimensions),
    ),
  };
}

function toCoreUISetComponent(
  component: UISetComponent,
  index: number,
  dimensions: { width: number; height: number } | undefined,
) {
  return {
    id: numericId(component.id, index + 1),
    name: component.label,
    size: {
      width: fromPercentage(component.bounds.width, dimensions?.width),
      height: fromPercentage(component.bounds.height, dimensions?.height),
    },
    position: {
      x: fromPercentage(component.bounds.x, dimensions?.width),
      y: fromPercentage(component.bounds.y, dimensions?.height),
    },
    metadata: { kind: component.kind },
  };
}

function readUISetComponentKind(
  metadata: Record<string, unknown> | undefined,
): UISetComponent["kind"] {
  const kind = readString(metadata, "kind");
  return kind === "label" || kind === "button" || kind === "panel"
    ? kind
    : "panel";
}

function readString(value: Record<string, unknown> | undefined, key: string) {
  const field = value?.[key];
  return typeof field === "string" ? field : undefined;
}

function toPercentage(value: number, total: number) {
  return total > 0 ? (value / total) * 100 : 0;
}

function fromPercentage(value: number, total: number | undefined) {
  return total && total > 0 ? (value / 100) * total : value;
}

function numericId(value: string, fallback: number) {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : fallback;
}
