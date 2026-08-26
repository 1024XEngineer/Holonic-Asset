import { useTranslation } from "react-i18next";

import type { GenerationTaskListItem } from "@/features/generation";
import type { AssetKind } from "@/model";

import { EditorHeader } from "../Header/editor-header";

export type EditorModeFrameProps = {
  assetKind: AssetKind;
  assetName: string;
  version: string;
  projectName: string;
  onBack: () => void;
  canExport?: boolean;
  isExporting?: boolean;
  onExport?: () => void;
  children: React.ReactNode;
};

const emptyGenerationTasks: GenerationTaskListItem[] = [];
const noAction = () => undefined;

export function EditorModeFrame({
  assetKind,
  assetName,
  version,
  projectName,
  onBack,
  canExport = false,
  isExporting = false,
  onExport = noAction,
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
        canExport={canExport}
        isExporting={isExporting}
        generationTasks={emptyGenerationTasks}
        onBack={onBack}
        onUndo={noAction}
        onRedo={noAction}
        onSave={noAction}
        onExport={onExport}
      />
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
        {children}
      </div>
    </>
  );
}
