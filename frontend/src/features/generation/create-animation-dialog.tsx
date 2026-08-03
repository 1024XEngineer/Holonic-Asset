import type { GenerateAnimationRequest } from "@/model";

type CreateAnimationDialogProps = {
  children: (openDialog: () => void) => React.ReactNode;
  isGenerating: boolean;
  onGenerate: (request: GenerateAnimationRequest) => void;
};

export function CreateAnimationDialog(_props: CreateAnimationDialogProps) {
  return null;
}
