type ExportFormat = "zip" | "spritesheet" | "tileset";
type ExportStatus = "queued" | "processing" | "ready" | "failed";
export type ExportSpecification = {
  projectId: string;
  assetIds: string[];
  format: ExportFormat;
  includeSourceFiles: boolean;
};
export type ExportJob = ExportSpecification & {
  id: string;
  status: ExportStatus;
  progress: number;
  downloadUrl?: string;
};
