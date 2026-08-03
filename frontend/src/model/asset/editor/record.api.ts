import { getMockEditorRecord, saveMockEditorRecordRevision } from "./mock";
import type { EditorRecordApi } from "./types";

export const recordApi: EditorRecordApi = {
  get: getMockEditorRecord,
  saveRevision: saveMockEditorRecordRevision,
};

export type {
  EditorRecordApi,
  EditorRecordSaveResult,
  GetEditorRecordInput,
  SaveEditorRecordInput,
} from "./types";
