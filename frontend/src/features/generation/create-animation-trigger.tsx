import type { GenerateAnimationRequest } from "@/model";

type CreateAnimationTriggerProps = {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  onGenerate: (request: GenerateAnimationRequest) => void;
};

export function CreateAnimationTrigger(_props: CreateAnimationTriggerProps) {
  return null;
}
