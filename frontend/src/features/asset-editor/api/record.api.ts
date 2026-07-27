import { getMockRecord, type GetRecordInput } from "./mock";
import type { RecordData } from "@/features/assets/domain";

export type RecordApi = {
  get: (input: GetRecordInput) => Promise<RecordData>;
};

export const recordApi: RecordApi = {
  get: getMockRecord,
};
