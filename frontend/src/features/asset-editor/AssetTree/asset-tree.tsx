import { ScrollArea } from "@/components/ui/scroll-area";
import { CreateAnimationDialog } from "@/features/generation/create-animation-dialog";
import type {
  EditorCharacterAnimation,
  GenerateAnimationRequest,
} from "@/model";
import { type AnimatedSpriteNodeId } from "../Canvas/AnimatedSpriteCanvas";
import { useAnimationActions } from "./animation-actions";
import { AnimationTreeItems } from "./animation-tree-items";

type AssetTreeProps = {
  animations: EditorCharacterAnimation[];
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onGenerateAnimation: (request: GenerateAnimationRequest) => void;
  onRenameAnimation: (animationId: string, label: string) => void;
  onDeleteAnimation: (animationId: string) => void;
  isGeneratingAnimation: boolean;
};

export function AssetTree({
  animations,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onGenerateAnimation,
  onRenameAnimation,
  onDeleteAnimation,
  isGeneratingAnimation,
}: AssetTreeProps) {
  const { openContextMenu, actions } = useAnimationActions({
    onRename: onRenameAnimation,
    onDelete: onDeleteAnimation,
  });

  return (
    <CreateAnimationDialog
      isGenerating={isGeneratingAnimation}
      onGenerate={onGenerateAnimation}
    >
      {(openGenerationDialog) => (
        <aside className="flex w-full shrink-0 flex-col border-b border-black/10 bg-[#ffffff] lg:h-full lg:w-[16.5rem] lg:border-b-0 lg:border-r">
          <ScrollArea className="max-h-[15rem] flex-1 lg:max-h-none">
            <div className="space-y-2 p-3">
              <AnimationTreeItems
                animations={animations}
                selectedNode={selectedNode}
                selectedFrames={selectedFrames}
                onSelect={onSelect}
                onSelectFrame={onSelectFrame}
                onOpenGeneration={openGenerationDialog}
                onOpenContextMenu={openContextMenu}
                isGenerating={isGeneratingAnimation}
              />
            </div>
          </ScrollArea>
          {actions}
        </aside>
      )}
    </CreateAnimationDialog>
  );
}
