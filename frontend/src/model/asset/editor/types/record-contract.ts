import type { AssetRevision } from "../../types";
import type { EditorRecord } from "./editor-record";
import type { EditorWorkspaceData } from "./editor-workspace";

export type GetEditorRecordInput = {
  projectId: string;
  assetId: string;
};

export type SaveEditorRecordInput = GetEditorRecordInput & {
  record: EditorRecord;
};

export type EditorRecordSaveResult = GetEditorRecordInput & {
  record: EditorRecord;
  version: string;
  history: AssetRevision[];
};

export type EditorRecordApi = {
  get: (input: GetEditorRecordInput) => Promise<EditorWorkspaceData>;
  saveRevision: (
    input: SaveEditorRecordInput,
  ) => Promise<EditorRecordSaveResult>;
};
