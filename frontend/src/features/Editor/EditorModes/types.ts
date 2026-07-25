import type { AssetRecord } from "@/types/record";

export type EditorModeProps = {
  prompt: string;
  history: AssetRecord[];
  onAction: (message: string) => void;
  onPromptChange: (value: string) => void;
  renderHeader: (selection: string) => React.ReactNode;
};
