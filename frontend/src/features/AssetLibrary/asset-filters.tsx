import { Funnel, X } from "lucide-react";

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

import { assetKinds, type AssetKind } from "@/domain/asset";
import { getAssetTypeConfig } from "@/components/ui/custom/asset-type-config";
import { AssetTypeIcon } from "@/components/ui/custom/AssetTypeIcon";

const filters: AssetKind[] = [...assetKinds];

export function AssetFilters({
  selectedKinds,
  onSelectedKindsChange,
}: {
  selectedKinds: AssetKind[];
  onSelectedKindsChange: (kinds: AssetKind[]) => void;
}) {
  const toggleKind = (kind: AssetKind, isSelected: boolean) => {
    onSelectedKindsChange(
      isSelected
        ? [...selectedKinds, kind]
        : selectedKinds.filter((selectedKind) => selectedKind !== kind),
    );
  };

  const activeFilterCount = filters.filter(
    (kind) => !selectedKinds.includes(kind),
  ).length;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            aria-label="Search and filter assets"
            size="icon"
            variant="outline"
            className="relative size-12 rounded-2xl bg-background shadow-none"
          />
        }
      >
        <Funnel />
        {activeFilterCount > 0 ? (
          <span className="absolute -right-1 -top-1 grid size-5 place-items-center rounded-full bg-primary text-[0.65rem] text-primary-foreground">
            {activeFilterCount}
          </span>
        ) : null}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel>Asset type</DropdownMenuLabel>
          {filters.map((kind) => (
            <DropdownMenuCheckboxItem
              key={kind}
              checked={selectedKinds.includes(kind)}
              closeOnClick={false}
              onCheckedChange={(checked) => toggleKind(kind, checked)}
            >
              <AssetTypeIcon kind={kind} />
              {getAssetTypeConfig(kind).label}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
        {activeFilterCount > 0 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => {
                onSelectedKindsChange(filters);
              }}
            >
              <X />
              Clear filters
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
