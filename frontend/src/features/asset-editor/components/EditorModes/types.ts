import type { AssetRecord } from "@/features/assets/domain";

export type EditorModeProps = {
  prompt: string;
  history: AssetRecord[];
  onAction: (message: string) => void;
  onPromptChange: (value: string) => void;
  renderHeader: (selection: string) => React.ReactNode;
};
