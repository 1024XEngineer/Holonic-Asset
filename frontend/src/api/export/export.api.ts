import { createMockExportJob, mockExportJobs } from "@/api/mock";
import type { ExportSpecification } from "@/domain/export";

export const exportApi = {
  list: async () => structuredClone(mockExportJobs),
  create: (specification: ExportSpecification) =>
    Promise.resolve(createMockExportJob(specification)),
};
