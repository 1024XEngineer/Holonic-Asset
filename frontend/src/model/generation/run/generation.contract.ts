import type { components, operations } from "@/model/generated/core-api";

type Schemas = components["schemas"];

export type GenerationTaskType = Schemas["CreateGenerationRequest"]["kind"];
export type GenerationTaskStatus =
  Schemas["GenerationRunListItemResponse"]["status"];
export type CreateGenerationRequest =
  operations["createGenerationRun"]["requestBody"]["content"]["application/json"];
export type CreateGenerationResponse =
  operations["createGenerationRun"]["responses"][200]["content"]["application/json"]["data"];
export type ListGenerationRunsQuery = NonNullable<
  operations["listGenerationRuns"]["parameters"]["query"]
>;
export type GenerationRunListItemResponse =
  Schemas["GenerationRunListItemResponse"];
export type ListGenerationRunsResponse =
  operations["listGenerationRuns"]["responses"][200]["content"]["application/json"]["data"];
type GeneratedGenerationRunResponse =
  operations["getGenerationRun"]["responses"][200]["content"]["application/json"]["data"];
type GeneratedGenerationResult = NonNullable<
  GeneratedGenerationRunResponse["result"]
>;

export type GenerationRunResponse<Content = unknown> = Omit<
  GeneratedGenerationRunResponse,
  "result"
> & {
  result?: Omit<GeneratedGenerationResult, "content"> & {
    content?: Content;
  };
};
export type CancelGenerationResponse =
  operations["cancelGenerationRun"]["responses"][200]["content"]["application/json"]["data"];
export type ResolveGenerationApplicationRequest =
  operations["resolveGenerationApplication"]["requestBody"]["content"]["application/json"];
