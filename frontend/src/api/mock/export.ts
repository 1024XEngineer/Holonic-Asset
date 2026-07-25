import type { ExportJob, ExportSpecification } from "@/types/export";

export const mockExportJobs: ExportJob[] = [
  {
    id: "export-moonlit-orchard-v4",
    projectId: "moonlit-orchard",
    assetIds: ["forager-hero", "orchard-ground-set"],
    format: "zip",
    includeSourceFiles: true,
    status: "ready",
    progress: 100,
    downloadUrl: "/downloads/moonlit-orchard-v4.zip",
  },
];

export function createMockExportJob(
  specification: ExportSpecification,
): ExportJob {
  return {
    ...specification,
    id: `export-${crypto.randomUUID()}`,
    status: "queued",
    progress: 0,
  };
}
