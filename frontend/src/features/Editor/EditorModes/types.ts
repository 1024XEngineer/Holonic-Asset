import type { AssetRecord } from "@/domain/asset";

export type EditorModeProps = {
  prompt: string;
  history: AssetRecord[];
  onAction: (message: string) => void;
  onPromptChange: (value: string) => void;
  renderHeader: (selection: string) => React.ReactNode;
};
