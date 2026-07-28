import type { AssetRevision } from "@/model";

export type EditorModeProps = {
  prompt: string;
  history: AssetRevision[];
  onAction: (message: string) => void;
  onPromptChange: (value: string) => void;
  renderHeader: (selection: string) => React.ReactNode;
};
