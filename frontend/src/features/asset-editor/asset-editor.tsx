import { useSuspenseRecordQuery } from "@/model";

import { EditorWorkspace } from "./editor-workspace";

export function AssetEditor({
  assetId,
  onExitEditor,
}: {
  assetId: string;
  onExitEditor: () => void;
}) {
  const { data } = useSuspenseRecordQuery(assetId);

  return <EditorWorkspace data={data} onBack={onExitEditor} />;
}
