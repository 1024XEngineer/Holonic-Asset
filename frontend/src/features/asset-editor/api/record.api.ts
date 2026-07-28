import { getMockEditorRecord, saveMockEditorRecordRevision } from "./mock";
import type { EditorRecordApi } from "./record.contract";

export const recordApi: EditorRecordApi = {
  get: getMockEditorRecord,
  saveRevision: saveMockEditorRecordRevision,
};

export type {
  EditorRecordApi,
  EditorRecordSaveResult,
  GetEditorRecordInput,
  SaveEditorRecordInput,
} from "./record.contract";
