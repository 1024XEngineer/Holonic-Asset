import { EditorWorkspace } from "./editor-workspace";

export function AssetEditor({
  assetId,
  onBack,
}: {
  assetId: string;
  onBack: () => void;
}) {
  return <EditorWorkspace assetId={assetId} onBack={onBack} />;
}
