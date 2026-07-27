import type { AssetRevision } from "@/features/assets/domain";

export type EditorModeProps = {
  prompt: string;
  history: AssetRevision[];
  onAction: (message: string) => void;
  onPromptChange: (value: string) => void;
  renderHeader: (selection: string) => React.ReactNode;
};
