import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useEffect, useRef, useState, type MouseEvent } from "react";
import { ChevronDown, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { AssetKindIcon, getAssetKindConfig } from "@/components/asset-kind";
import type { CreatableAssetKind } from "@/model/asset";
import type { ProjectSummary } from "@/model/project";

export function CreateAssetToolbar({
  assetKinds,
  project,
}: {
  assetKinds: CreatableAssetKind[];
  project: ProjectSummary;
}) {
  const { t } = useTranslation("workspace");
  const navigate = useNavigate();
  const [hovered, setHovered] = useState(false);
  const [pinned, setPinned] = useState(false);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );
  useEffect(
    () => () => {
      if (closeTimer.current) clearTimeout(closeTimer.current);
    },
    [],
  );
  const isOpen = hovered || pinned;
  const keepOpen = () => {
    if (closeTimer.current) clearTimeout(closeTimer.current);
    setHovered(true);
  };
  const scheduleClose = () => {
    closeTimer.current = setTimeout(() => setHovered(false), 150);
  };
  const handleMenuLeave = (event: MouseEvent<HTMLDivElement>) => {
    const nextTarget = event.relatedTarget;
    if (
      nextTarget instanceof Node &&
      event.currentTarget.parentElement?.contains(nextTarget)
    ) {
      return;
    }
    scheduleClose();
  };

  return (
    <div
      className="relative"
      onMouseEnter={keepOpen}
      onMouseLeave={scheduleClose}
    >
      <Button
        type="button"
        className="h-9 bg-black px-4 text-white hover:bg-black/80"
        aria-expanded={isOpen}
        aria-haspopup="menu"
        onClick={() => setPinned((current) => !current)}
      >
        <Plus data-icon="inline-start" />
        {t("generation.newAsset")}
        <ChevronDown
          className={`size-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
      </Button>
      {isOpen ? (
        <div
          className="absolute top-full right-0 z-50 pt-2"
          onMouseEnter={keepOpen}
          onMouseLeave={handleMenuLeave}
        >
          <div
            className="grid w-56 gap-1 rounded-lg border bg-background p-1 shadow-lg"
            role="menu"
          >
            {assetKinds.map((kind) => (
              <Button
                key={kind}
                type="button"
                variant="ghost"
                role="menuitem"
                className="h-9 justify-start px-3 text-sm"
                onClick={() => {
                  setPinned(false);
                  setHovered(false);
                  void navigate({
                    to: "/projects/$projectId/assets/new/$kind",
                    params: { projectId: project.id, kind },
                  });
                }}
              >
                <AssetKindIcon kind={kind} className="size-4" />
                {t("generation.createKind", {
                  kind: getAssetKindConfig(kind).label,
                })}
              </Button>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}
