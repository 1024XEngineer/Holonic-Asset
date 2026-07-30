export type AssetRevisionStatus = "ready" | "generating" | "failed";

export type AssetRevision<Payload = unknown> = {
  id: string;
  version: string;
  description: string;
  savedAt?: string;
  status: AssetRevisionStatus;
  isCurrent: boolean;
  content?: Payload;
};
