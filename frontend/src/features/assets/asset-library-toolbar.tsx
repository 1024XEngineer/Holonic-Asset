import { Plus, Search, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import { AssetFilters } from "./asset-filters";
import type { AssetLibraryController } from "./state/use-asset-library-controller";

export function AssetLibraryToolbar({
  library,
}: {
  library: AssetLibraryController;
}) {
  return (
    <div className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 border-b pb-5">
      <AssetFilters
        counts={library.counts}
        selectedKinds={library.selectedKinds}
        onSelectedKindsChange={library.setSelectedKinds}
      />

      <div className="relative min-w-0">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          aria-label="Search assets"
          className="h-9 bg-background pl-9 pr-9"
          placeholder="Search names, tags, or descriptions"
          type="search"
          value={library.query}
          onChange={(event) => library.setQuery(event.target.value)}
        />
        {library.query ? (
          <Button
            type="button"
            aria-label="Clear search"
            title="Clear search"
            variant="ghost"
            size="icon-xs"
            className="absolute right-1.5 top-1/2 -translate-y-1/2 text-muted-foreground"
            onClick={() => library.setQuery("")}
          >
            <X />
          </Button>
        ) : null}
      </div>
      <Button type="button" className="h-9" title="Create a new asset">
        <Plus data-icon="inline-start" />
        New asset
      </Button>
    </div>
  );
}
