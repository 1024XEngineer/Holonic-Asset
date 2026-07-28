import type { AssetRevision } from "@/model";
import type { EditorRecord, EditorWorkspaceData } from "@/model";

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
