import {
  ChevronDown,
  Eye,
  EyeOff,
  Folder,
  ImagePlus,
  Layers3,
  Play,
  Plus,
} from "lucide-react";
import { useState, type MouseEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { CreateAnimationTrigger } from "@/features/generation";
import type {
  CharacterAnimation,
  GenerateAnimationRequest,
  Perspective,
  SceneryLayer,
} from "@/model";
import { ScrollArea } from "@/components/ui/scroll-area";

import {
  getAnimatedSpriteNodeLabel,
  type AnimatedSpriteNodeId,
} from "../Canvas/AnimatedSpriteCanvas";
import { useAnimationActions } from "./animation-actions";

type SpriteAssetTreeProps = {
  kind: "sprite";
  animations: CharacterAnimation[];
  perspective: Perspective;
  selectedNode: AnimatedSpriteNodeId | null;
  selectedFrames: Array<{ nodeId: AnimatedSpriteNodeId; index: number }>;
  onSelect: (node: AnimatedSpriteNodeId) => void;
  onSelectFrame: (node: AnimatedSpriteNodeId, index: number) => void;
  onGenerateAnimation: (request: GenerateAnimationRequest) => void;
  onRenameAnimation: (animationId: string, label: string) => void;
  onDeleteAnimation: (animationId: string) => void;
  isGeneratingAnimation: boolean;
};

type SceneryAssetTreeProps = {
  kind: "scenery";
  layers: readonly SceneryLayer[];
  selectedLayerId: string | null;
  visibleLayerIds: readonly string[];
  onSelect: (layerId: string) => void;
  onToggleVisibility: (layerId: string) => void;
};

type AssetTreeProps = SpriteAssetTreeProps | SceneryAssetTreeProps;

export function AssetTree(props: AssetTreeProps) {
  return props.kind === "scenery" ? (
    <SceneryAssetTree {...props} />
  ) : (
    <SpriteAssetTree {...props} />
  );
}

function SpriteAssetTree({
  kind: _kind,
  animations,
  perspective,
  selectedNode,
  selectedFrames,
  onSelect,
  onSelectFrame,
  onGenerateAnimation,
  onRenameAnimation,
  onDeleteAnimation,
  isGeneratingAnimation,
}: SpriteAssetTreeProps) {
  const { t } = useTranslation("editor");
  const { openContextMenu, actions } = useAnimationActions({
    onRename: onRenameAnimation,
    onDelete: onDeleteAnimation,
  });

  return (
    <CreateAnimationTrigger
      isGenerating={isGeneratingAnimation}
      perspective={perspective}
      onGenerate={onGenerateAnimation}
    >
      {(openGenerationDialog) => (
        <aside className="flex w-full shrink-0 flex-col border-b bg-background lg:h-full lg:w-64 lg:border-b-0 lg:border-r">
          <div className="flex items-center justify-between border-b px-4 py-3">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                {t("assetTree")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("assetTreeDescription")}
              </p>
            </div>
            <span className="font-mono text-[10px] text-muted-foreground">
              {animations.length + 1}
            </span>
          </div>
          <ScrollArea className="max-h-56 flex-1 lg:max-h-none">
            <div className="space-y-2 p-3">
              <TreeNode
                node="prototype"
                selected={selectedNode === "prototype"}
                onSelect={onSelect}
                icon={<ImagePlus className="size-4" />}
              />
              <div>
                <div className="flex items-center gap-2 px-2 py-2 text-xs font-medium text-muted-foreground">
                  <Folder className="size-4 text-primary" />
                  <span className="min-w-0 flex-1 truncate">
                    {t("animations")}
                  </span>
                  <span className="font-mono text-[10px]">
                    {animations.length}
                  </span>
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
                    />
                  ))}
                </div>
              </div>
            </div>
          </ScrollArea>
          {actions}
        </aside>
      )}
    </CreateAnimationTrigger>
  );
}

function SceneryAssetTree({
  layers,
  selectedLayerId,
  visibleLayerIds,
  onSelect,
  onToggleVisibility,
}: SceneryAssetTreeProps) {
  const { t } = useTranslation("editor");
  const visible = new Set(visibleLayerIds);

  return (
    <aside className="flex w-full shrink-0 flex-col border-b bg-background lg:h-full lg:w-64 lg:border-b-0 lg:border-r">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">
            {t("sceneryLayers")}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("sceneryLayersDescription")}
          </p>
        </div>
        <span className="font-mono text-[10px] text-muted-foreground">
          {layers.length}
        </span>
      </div>
      <ScrollArea className="max-h-56 flex-1 lg:max-h-none">
        <div className="space-y-1 p-3">
          {[...layers].reverse().map((layer, index) => {
            const isVisible = visible.has(layer.id);
            const selected = selectedLayerId === layer.id;
            return (
              <div
                key={layer.id}
                className={`flex items-center gap-1 rounded-lg border transition-colors ${selected ? "border-primary/30 bg-primary/10" : "border-transparent hover:bg-muted"}`}
              >
                <button
                  type="button"
                  aria-current={selected ? "true" : undefined}
                  onClick={() => onSelect(layer.id)}
                  className="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
                >
                  <Layers3 className="size-4 shrink-0 text-primary" />
                  <span className="min-w-0 flex-1 truncate text-xs font-medium">
                    {layer.label}
                  </span>
                  {index === layers.length - 1 ? (
                    <span className="text-[10px] text-muted-foreground">
                      {t("backdrop")}
                    </span>
                  ) : null}
                </button>
                <button
                  type="button"
                  aria-label={
                    isVisible
                      ? t("hideLayer", { value: layer.label })
                      : t("showLayer", { value: layer.label })
                  }
                  aria-pressed={isVisible}
                  title={isVisible ? t("hide") : t("show")}
                  onClick={() => onToggleVisibility(layer.id)}
                  className="mr-1 grid size-7 place-items-center rounded-md text-muted-foreground hover:bg-background hover:text-foreground"
                >
                  {isVisible ? (
                    <Eye className="size-3.5" />
                  ) : (
                    <EyeOff className="size-3.5" />
                  )}
                </button>
              </div>
            );
          })}
          {layers.length === 0 ? (
            <div className="px-2 py-8 text-center text-xs text-muted-foreground">
              {t("noSceneryLayers")}
            </div>
          ) : null}
        </div>
      </ScrollArea>
    </aside>
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
