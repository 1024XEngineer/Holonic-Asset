import { useTranslation } from "react-i18next";

import type { AssetKind } from "@/model";

import {
  EditorHeader,
  type EditorGenerationTask,
} from "../Header/editor-header";

export type EditorModeFrameProps = {
  assetKind: AssetKind;
  assetName: string;
  version: string;
  projectName: string;
  onBack: () => void;
  children: React.ReactNode;
};

const emptyGenerationTasks: EditorGenerationTask[] = [];
const noAction = () => undefined;

export function EditorModeFrame({
  assetKind,
  assetName,
  version,
  projectName,
  onBack,
  children,
}: EditorModeFrameProps) {
  const { t } = useTranslation("editor");
  return (
    <>
      <EditorHeader
        assetKind={assetKind}
        assetName={assetName}
        version={version}
        projectName={projectName}
        status={t("previewReady")}
        canUndo={false}
        canRedo={false}
        isDirty={false}
        isSaving={false}
        generationTasks={emptyGenerationTasks}
        onBack={onBack}
        onUndo={noAction}
        onRedo={noAction}
        onSave={noAction}
      />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        {children}
      </div>
    </>
  );
}
