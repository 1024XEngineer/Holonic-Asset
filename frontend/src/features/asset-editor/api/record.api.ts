import { getMockEditorDocument, saveMockEditorDocumentRevision } from "./mock";
import type { EditorDocumentApi } from "./record.contract";

export const recordApi: EditorDocumentApi = {
  get: getMockEditorDocument,
  saveRevision: saveMockEditorDocumentRevision,
};

export type {
  EditorDocumentApi,
  EditorDocumentSaveResult,
  GetEditorDocumentInput,
  SaveEditorDocumentInput,
} from "./record.contract";
