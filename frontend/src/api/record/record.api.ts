import { getMockRecord, type GetRecordInput } from "@/api/mock";
import type { RecordData } from "@/types/record";

export type RecordApi = {
  get: (input: GetRecordInput) => Promise<RecordData>;
};

export const recordApi: RecordApi = {
  get: getMockRecord,
};
