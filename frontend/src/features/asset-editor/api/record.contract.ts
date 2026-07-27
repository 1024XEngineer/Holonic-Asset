import type { AssetRevision } from "@/features/assets/domain";
import type { EditorDocument, EditorWorkspaceData } from "../domain";

export type GetEditorDocumentInput = {
  projectId: string;
  assetId: string;
};

export type SaveEditorDocumentInput = GetEditorDocumentInput & {
  content: EditorDocument;
};

export type EditorDocumentSaveResult = GetEditorDocumentInput & {
  content: EditorDocument;
  version: string;
  history: AssetRevision[];
};

export type EditorDocumentApi = {
  get: (input: GetEditorDocumentInput) => Promise<EditorWorkspaceData>;
  saveRevision: (
    input: SaveEditorDocumentInput,
  ) => Promise<EditorDocumentSaveResult>;
};
