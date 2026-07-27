import { createMockExportJob, mockExportJobs } from "./mock/export";
import type { ExportSpecification } from "@/features/export/domain";

export const exportApi = {
  list: async () => structuredClone(mockExportJobs),
  create: (specification: ExportSpecification) =>
    Promise.resolve(createMockExportJob(specification)),
};
