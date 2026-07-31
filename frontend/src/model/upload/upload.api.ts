import type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";
import { postEnvelope } from "@/model/fetchers";

export type {
  CreateUploadTargetRequest,
  UploadTarget,
} from "./upload.contract";

export const uploadApi = {
  createTarget: (request: CreateUploadTargetRequest) =>
    postEnvelope<UploadTarget>("/uploads", request),
};
