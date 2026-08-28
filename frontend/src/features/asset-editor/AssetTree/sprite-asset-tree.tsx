import {
  ChevronDown,
  Folder,
  GitFork,
  ImagePlus,
  Play,
  Plus,
} from "lucide-react";
import { useState, type MouseEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { CreateAnimationTrigger } from "@/features/generation";
import {
  assetDirectionsByPerspective,
  type AssetDirection,
  type CharacterAnimation,
  type DeriveAnimationRequest,
  type GenerateAnimationRequest,
  type Perspective,
} from "@/model";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import { useAnimationActions } from "./animation-actions";
import { AssetTree } from "./asset-tree";

export function SpriteAssetTree({
  animations,
  prototypeDimensions,
  perspective,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onGenerateAnimation,
  onDeriveAnimation,
  onRenameAnimation,
  onDeleteAnimation,
  isGeneratingAnimation,
}: {
  animations: CharacterAnimation[];
  prototypeDimensions?: { width: number; height: number };
  perspective: Perspective;
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onGenerateAnimation: (request: GenerateAnimationRequest) => void;
  onDeriveAnimation: (request: DeriveAnimationRequest) => void;
  onRenameAnimation: (animationId: string, label: string) => void;
  onDeleteAnimation: (animationId: string) => void;
  isGeneratingAnimation: boolean;
}) {
  const { t } = useTranslation("editor");
  const { openContextMenu, actions } = useAnimationActions({
    onRename: onRenameAnimation,
    onDelete: onDeleteAnimation,
  });

  return (
    <CreateAnimationTrigger
      isGenerating={isGeneratingAnimation}
      prototypeDimensions={prototypeDimensions}
      perspective={perspective}
      onGenerate={onGenerateAnimation}
    >
      {(openGenerationDialog) => (
        <AssetTree
          title={t("assetTree")}
          description={t("assetTreeDescription")}
          count={animations.length + 1}
          footer={actions}
          contentClassName="space-y-2"
        >
          <TreeNode
            node="prototype"
            selected={selectedNode === "prototype"}
            onSelect={onSelect}
            icon={<ImagePlus className="size-4" />}
          />
          <div>
            <div className="flex items-center gap-2 px-2 py-2 text-xs font-medium text-muted-foreground">
              <Folder className="size-4 text-primary" />
              <span className="min-w-0 flex-1 truncate">{t("animations")}</span>
              <span className="font-mono text-[10px]">{animations.length}</span>
              <button
                type="button"
                aria-label={t("generateAnimation")}
                title={t("generateAnimation")}
                disabled={isGeneratingAnimation}
                onClick={openGenerationDialog}
                className="grid size-6 place-items-center rounded-md border border-dashed text-muted-foreground transition-colors hover:border-primary/60 hover:bg-primary/5 hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              >
                <Plus className="size-3.5" />
              </button>
            </div>
            <div className="ml-4 space-y-0.5 border-l pl-2">
              {animations.map((animation) => (
                <AnimationNode
                  key={animation.id}
                  animation={animation}
                  selected={selectedNode === animation.id}
                  selectedFrames={selectedFrames}
                  onSelect={onSelect}
                  onSelectFrame={onSelectFrame}
                  onContextMenu={openContextMenu}
                  perspective={perspective}
                  animations={animations}
                  isGenerating={isGeneratingAnimation}
                  onDerive={onDeriveAnimation}
                />
              ))}
            </div>
          </div>
        </AssetTree>
      )}
    </CreateAnimationTrigger>
  );
}

function TreeNode({
  node,
  selected,
  onSelect,
  icon,
}: {
  node: AnimatedSpriteNodeId;
  selected: boolean;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  icon: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(node)}
      className={`group flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left transition-colors ${selected ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
    >
      <span className="text-primary">{icon}</span>
      <span className="min-w-0 flex-1 truncate text-xs font-medium">
        {getAnimatedSpriteNodeLabel(node, [])}
      </span>
    </button>
  );
}

function AnimationNode({
  animation,
  selected,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onContextMenu,
  perspective,
  animations,
  isGenerating,
  onDerive,
}: {
  animation: CharacterAnimation;
  selected: boolean;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onContextMenu: (
    event: MouseEvent<HTMLElement>,
    animation: CharacterAnimation,
  ) => void;
  perspective: Perspective;
  animations: CharacterAnimation[];
  isGenerating: boolean;
  onDerive: (request: DeriveAnimationRequest) => void;
}) {
  const [open, setOpen] = useState(false);
  const { t } = useTranslation("editor");

  return (
    <div onContextMenu={(event) => onContextMenu(event, animation)}>
      <div
        className={`flex items-center rounded-lg transition-colors ${selected ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
      >
        <button
          type="button"
          onClick={() => onSelect(animation.id)}
          className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
        >
          <Play className="size-3.5 text-emerald-600" />
          <span className="min-w-0 flex-1 truncate text-xs font-medium">
            {animation.label}
          </span>
        </button>
        <DeriveAnimationButton
          animation={animation}
          animations={animations}
          perspective={perspective}
          isGenerating={isGenerating}
          onDerive={onDerive}
        />
        <button
          type="button"
          aria-label={`${open ? t("collapse") : t("expand")} ${animation.label}`}
          aria-expanded={open}
          onClick={() => setOpen((current) => !current)}
          className="mr-1 rounded-md p-1.5 text-muted-foreground hover:bg-muted"
        >
          <ChevronDown
            className={`size-3.5 transition-transform ${open ? "rotate-0" : "-rotate-90"}`}
          />
        </button>
      </div>
      {open ? (
        <AnimationFrames
          animation={animation}
          selectedFrames={selectedFrames}
          onSelectFrame={onSelectFrame}
        />
      ) : null}
    </div>
  );
}

function DeriveAnimationButton({
  animation,
  animations,
  perspective,
  isGenerating,
  onDerive,
}: {
  animation: CharacterAnimation;
  animations: CharacterAnimation[];
  perspective: Perspective;
  isGenerating: boolean;
  onDerive: (request: DeriveAnimationRequest) => void;
}) {
  const { t } = useTranslation(["editor", "generation"]);
  const [selectedDirections, setSelectedDirections] = useState<
    AssetDirection[]
  >([]);
  const availableDirections = getAvailableDerivationDirections(
    animation,
    animations,
    perspective,
  );
  const canDerive =
    animation.generation !== undefined && availableDirections.length > 0;

  const toggleDirection = (direction: AssetDirection) => {
    setSelectedDirections((current) =>
      current.includes(direction)
        ? current.filter((value) => value !== direction)
        : [...current, direction],
    );
  };

  return (
    <Popover
      onOpenChange={(open) => {
        if (open) setSelectedDirections([]);
      }}
    >
      <PopoverTrigger
        aria-label={t("deriveAnimation", { name: animation.label })}
        title={t("deriveAnimation", { name: animation.label })}
        disabled={!canDerive || isGenerating}
        className="grid size-6 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-primary disabled:pointer-events-none disabled:opacity-40"
      >
        <GitFork className="size-3.5" />
      </PopoverTrigger>
      <PopoverContent align="end" className="w-56 space-y-3">
        <div>
          <p className="text-sm font-medium">{t("deriveAnimationTitle")}</p>
          <p className="text-xs text-muted-foreground">
            {t("deriveAnimationDescription")}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-1">
          {availableDirections.map((direction) => {
            const selected = selectedDirections.includes(direction);
            return (
              <button
                key={direction}
                type="button"
                aria-pressed={selected}
                onClick={() => toggleDirection(direction)}
                className={`rounded-md border px-2 py-1.5 text-xs font-medium transition-colors ${selected ? "border-primary bg-primary/10 text-primary" : "border-transparent bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground"}`}
              >
                {t(`generation:directions.${direction}`)}
              </button>
            );
          })}
        </div>
        <Button
          type="button"
          size="sm"
          className="w-full"
          disabled={selectedDirections.length === 0 || isGenerating}
          onClick={() =>
            onDerive({
              sourceAnimationId: animation.id,
              sourceAnimationName: animation.label,
              targetDirections: selectedDirections,
            })
          }
        >
          <GitFork className="size-3.5" />
          {t("deriveAnimationSubmit")}
        </Button>
      </PopoverContent>
    </Popover>
  );
}

function getAvailableDerivationDirections(
  source: CharacterAnimation,
  animations: CharacterAnimation[],
  perspective: Perspective,
) {
  const groupId = source.groupId ?? source.id;
  const occupiedDirections = new Set(
    animations.flatMap((animation) => {
      const belongsToGroup =
        animation.id === source.id ||
        animation.id === groupId ||
        animation.groupId === groupId;
      const direction = animation.generation?.direction as
        | AssetDirection
        | undefined;
      return belongsToGroup && direction ? [direction] : [];
    }),
  );
  return assetDirectionsByPerspective[perspective].filter(
    (direction) => !occupiedDirections.has(direction),
  );
}

function AnimationFrames({
  animation,
  selectedFrames,
  onSelectFrame,
}: {
  animation: CharacterAnimation;
  selectedFrames: Array<{
    nodeId: AnimatedSpriteNodeId;
    index: number;
  }>;
  onSelectFrame: (nodeId: AnimatedSpriteNodeId, index: number) => void;
}) {
  const { t } = useTranslation("editor");
  const selectedFrameIndexes = new Set(
    selectedFrames
      .filter((frame) => frame.nodeId === animation.id)
      .map((frame) => frame.index),
  );

  return (
    <div className="ml-4 mt-1 space-y-0.5 border-l pl-2">
      {Array.from({ length: animation.frameCount }, (_, index) => index).map(
        (index) => {
          const isSelected = selectedFrameIndexes.has(index);
          return (
            <button
              key={`${animation.id}-${index}`}
              type="button"
              aria-pressed={isSelected}
              onClick={() => onSelectFrame(animation.id, index)}
              className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[11px] transition-colors ${isSelected ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
            >
              <span className="size-1.5 rounded-full bg-current opacity-70" />
              {t("frame", { index: index + 1 })}
            </button>
          );
        },
      )}
    </div>
  );
}
