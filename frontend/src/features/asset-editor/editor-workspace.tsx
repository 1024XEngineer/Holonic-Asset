import { AlertTriangle, LoaderCircle } from "lucide-react";
import { useParams } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";
import { useRecordQuery } from "@/model";
import type { AssetWorkspaceData } from "@/model";

import { SpriteEditorMode } from "./EditorModes/sprite-editor-mode";
import { useEditorWorkspace } from "./use-editor-workspace";

export function EditorWorkspace({
  assetId,
  onBack,
}: {
  assetId: string;
  onBack: () => void;
}) {
  const { projectId } = useParams({
    from: "/projects/$projectId/assets/$assetId",
  });
  const recordQuery = useRecordQuery(projectId, assetId);

  if (recordQuery.isPending) {
    return (
      <EditorStatus
        icon={<LoaderCircle className="size-5 animate-spin" />}
        title="Loading asset editor"
        description="Preparing the latest asset record."
      />
    );
  }

  if (recordQuery.error) {
    return (
      <EditorStatus
        icon={<AlertTriangle className="size-5" />}
        title="Unable to open asset"
        description={recordQuery.error.message}
        action={
          <>
            <Button
              variant="outline"
              onClick={() => void recordQuery.refetch()}
            >
              Try again
            </Button>
            <Button onClick={onBack}>Back to library</Button>
          </>
        }
      />
    );
  }

  if (
    recordQuery.data.record.mode !== "character" &&
    recordQuery.data.record.mode !== "object"
  ) {
    return (
      <EditorStatus
        icon={<AlertTriangle className="size-5" />}
        title="Editor unavailable"
        description="This editor currently supports character and object assets."
        action={<Button onClick={onBack}>Back to library</Button>}
      />
    );
  }

  return <EditorWorkspaceContent data={recordQuery.data} onBack={onBack} />;
}

function EditorWorkspaceContent({
  data,
  onBack,
}: {
  data: AssetWorkspaceData;
  onBack: () => void;
}) {
  const editorProps = useEditorWorkspace({ data, onBack });
  if (!editorProps) return null;

  return (
    <div className="flex h-dvh min-h-0 w-full flex-col overflow-hidden bg-muted/30 text-foreground selection:bg-primary/20">
      <SpriteEditorMode {...editorProps} />
    </div>
  );
}

function EditorStatus({
  icon,
  title,
  description,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <main className="grid min-h-dvh place-items-center bg-muted/30 p-6">
      <section className="w-full max-w-md border bg-background p-6 shadow-sm">
        <div className="grid size-10 place-items-center border bg-muted text-muted-foreground">
          {icon}
        </div>
        <h1 className="mt-5 text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-muted-foreground">
          {description}
        </p>
        {action ? <div className="mt-6 flex gap-2">{action}</div> : null}
      </section>
    </main>
  );
}
