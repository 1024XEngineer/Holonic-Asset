import { Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CreateAssetToolbar } from "@/features/generation";
import { creatableAssetKinds } from "@/model/asset";

import { AssetFilters } from "./asset-filters";
import type { AssetLibraryController } from "./state/use-asset-library-controller";

export function AssetLibraryToolbar({
  library,
}: {
  library: AssetLibraryController;
}) {
  const { t } = useTranslation("workspace");
  return (
    <div className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-3 border-b pb-5 md:grid-cols-[auto_minmax(0,1fr)_auto] md:items-center">
      <AssetFilters
        counts={library.counts}
        selectedKinds={library.selectedKinds}
        onSelectedKindsChange={library.setSelectedKinds}
      />

      <div className="relative min-w-0">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          aria-label={t("assets.search")}
          className="h-9 bg-background pl-9 pr-9"
          placeholder={t("assets.searchPlaceholder")}
          type="search"
          value={library.query}
          onChange={(event) => library.setQuery(event.target.value)}
        />
        {library.query ? (
          <Button
            type="button"
            aria-label={t("assets.clearSearch")}
            title={t("assets.clearSearch")}
            variant="ghost"
            size="icon-xs"
            className="absolute right-1.5 top-1/2 -translate-y-1/2 text-muted-foreground"
            onClick={() => library.setQuery("")}
          >
            <X />
          </Button>
        ) : null}
      </div>
      {library.project ? (
        <CreateAssetToolbar
          assetKinds={creatableAssetKinds}
          project={library.project}
        />
      ) : null}
    </div>
  );
}
