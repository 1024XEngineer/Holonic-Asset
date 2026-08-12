import { RotateCcw, SlidersHorizontal } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AssetKindIcon } from "@/components/asset-kind";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { assetKinds, type AssetKind } from "@/model/asset";

export function AssetFilters({
  counts,
  selectedKinds,
  onSelectedKindsChange,
}: {
  counts: Record<AssetKind, number>;
  selectedKinds: AssetKind[];
  onSelectedKindsChange: (kinds: AssetKind[]) => void;
}) {
  const { t } = useTranslation(["assets", "common"]);
  const hiddenTypeCount = assetKinds.length - selectedKinds.length;

  const toggleKind = (kind: AssetKind, checked: boolean) => {
    if (checked && !selectedKinds.includes(kind)) {
      onSelectedKindsChange([...selectedKinds, kind]);
      return;
    }

    if (!checked) {
      onSelectedKindsChange(
        selectedKinds.filter((selectedKind) => selectedKind !== kind),
      );
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="icon-lg"
            aria-label={t("filter")}
            title={t("filter")}
          />
        }
      >
        <SlidersHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("assetType")}</DropdownMenuLabel>
          {assetKinds.map((kind) => (
            <DropdownMenuCheckboxItem
              key={kind}
              checked={selectedKinds.includes(kind)}
              closeOnClick={false}
              onCheckedChange={(checked) => toggleKind(kind, checked)}
            >
              <AssetKindIcon kind={kind} />
              <span className="flex-1">{t(`common:assetKinds.${kind}`)}</span>
              <span className="text-xs text-muted-foreground">
                {counts[kind]}
              </span>
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
        {hiddenTypeCount > 0 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => onSelectedKindsChange([...assetKinds])}
            >
              <RotateCcw />
              {t("showAll")}
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
