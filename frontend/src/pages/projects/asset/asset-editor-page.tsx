import { useNavigate, useParams } from "@tanstack/react-router";

import { AssetEditor } from "@/features/asset-editor";

export function AssetEditorPage() {
  const { assetId, projectId } = useParams({
    from: "/projects/$projectId/assets/$assetId",
  });
  const navigate = useNavigate({
    from: "/projects/$projectId/assets/$assetId",
  });

  return (
    <AssetEditor
      assetId={assetId}
      onBack={() =>
        void navigate({
          to: "/projects/$projectId",
          params: { projectId },
        })
      }
    />
  );
}
