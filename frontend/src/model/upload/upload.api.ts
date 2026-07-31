import type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";
import { postJson } from "@/model/fetchers";

export type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";

export const uploadApi = {
  createTarget: (request: CreateUploadTargetRequest) =>
    postJson<UploadTarget>("/uploads", request),
};
