import {
  ArrowLeft,
  Check,
  Dices,
  Pencil,
  Plus,
  Search,
  Tags,
  X,
} from "lucide-react";
import { useId, useMemo, useState } from "react";
import { HexColorPicker } from "react-colorful";
import { useTranslation } from "react-i18next";

import { AssetTagBadge } from "@/components/asset-tag-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverClose,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import {
  defaultAssetTagColor,
  getRandomAssetTagColor,
  mergeAssetTags,
  normalizeAssetTag,
  presetAssetTagColors,
  type AssetTag,
} from "@/model/asset";

export function AssetTagPicker({
  availableTags = [],
  className,
  disabled = false,
  id,
  onChange,
  tags,
}: {
  availableTags?: readonly AssetTag[];
  className?: string;
  disabled?: boolean;
  id: string;
  onChange: (tags: AssetTag[]) => void;
  tags: readonly AssetTag[];
}) {
  const { t } = useTranslation("common");
  const fallbackId = useId();
  const inputId = id || fallbackId;

  // Popover state
  const [isOpen, setIsOpen] = useState(false);
  const [view, setView] = useState<"list" | "create" | "edit">("list");
  const [filterQuery, setFilterQuery] = useState("");

  // Locally created or edited tags in this session
  const [localTags, setLocalTags] = useState<AssetTag[]>([]);

  // Form state for create / edit
  const [editingTarget, setEditingTarget] = useState<AssetTag | null>(null);
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formColor, setFormColor] = useState(defaultAssetTagColor);
  const [showColorPicker, setShowColorPicker] = useState(false);

  // Merge available, current, and locally created tags
  const allOptions = useMemo(
    () => mergeAssetTags(availableTags, localTags, tags),
    [availableTags, localTags, tags],
  );

  const selectedNames = useMemo(
    () => new Set(tags.map((tag) => tag.name.toLocaleLowerCase())),
    [tags],
  );

  // Filtered list
  const filteredOptions = useMemo(() => {
    const query = filterQuery.trim().toLocaleLowerCase();
    if (!query) return allOptions;
    return allOptions.filter(
      (tag) =>
        tag.name.toLocaleLowerCase().includes(query) ||
        tag.description.toLocaleLowerCase().includes(query),
    );
  }, [allOptions, filterQuery]);

  const hasExactMatch = useMemo(() => {
    const query = filterQuery.trim().toLocaleLowerCase();
    if (!query) return true;
    return allOptions.some((tag) => tag.name.toLocaleLowerCase() === query);
  }, [allOptions, filterQuery]);

  const selectTag = (tag: AssetTag) => {
    if (selectedNames.has(tag.name.toLocaleLowerCase())) return;
    onChange(mergeAssetTags(tags, [tag]));
  };

  const removeTag = (tag: AssetTag) => {
    const name = tag.name.toLocaleLowerCase();
    onChange(tags.filter((item) => item.name.toLocaleLowerCase() !== name));
  };

  const toggleTag = (tag: AssetTag) => {
    if (selectedNames.has(tag.name.toLocaleLowerCase())) {
      removeTag(tag);
    } else {
      selectTag(tag);
    }
  };

  const openCreateView = (initialName = "") => {
    setEditingTarget(null);
    setFormName(initialName);
    setFormDescription("");
    setFormColor(getRandomAssetTagColor());
    setShowColorPicker(false);
    setView("create");
  };

  const openEditView = (tag: AssetTag, event?: React.MouseEvent) => {
    event?.stopPropagation();
    setEditingTarget(tag);
    setFormName(tag.name);
    setFormDescription(tag.description);
    setFormColor(tag.color);
    setShowColorPicker(false);
    setView("edit");
  };

  const handleSaveForm = () => {
    const normalized = normalizeAssetTag({
      name: formName,
      description: formDescription,
      color: formColor,
    });
    if (!normalized) return;

    if (view === "edit" && editingTarget) {
      const oldNameLower = editingTarget.name.toLocaleLowerCase();
      // Update in selected tags if present
      const updatedSelected = tags.map((item) =>
        item.name.toLocaleLowerCase() === oldNameLower ? normalized : item,
      );
      // Update in local options
      setLocalTags((prev) =>
        prev.map((item) =>
          item.name.toLocaleLowerCase() === oldNameLower ? normalized : item,
        ),
      );
      onChange(updatedSelected);
    } else {
      // Create new tag
      setLocalTags((prev) => mergeAssetTags(prev, [normalized]));
      selectTag(normalized);
    }

    setFilterQuery("");
    setView("list");
  };

  const previewTag: AssetTag = useMemo(
    () => ({
      name: formName.trim() || "tag-preview",
      description: formDescription.trim(),
      color: formColor,
    }),
    [formName, formDescription, formColor],
  );

  return (
    <Popover
      open={isOpen}
      onOpenChange={(open) => {
        setIsOpen(open);
        if (!open) {
          setView("list");
          setFilterQuery("");
        }
      }}
    >
      <div className={cn("space-y-2", className)}>
        <div className="flex items-center">
          <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            <Tags className="size-3.5" aria-hidden="true" />
            <span id={`${inputId}-label`}>{t("tags.label")}</span>
          </div>
        </div>

        <PopoverContent
          align="end"
          sideOffset={6}
          className="w-80 p-0 text-xs shadow-lg sm:w-88"
        >
          {view === "list" ? (
            <div className="flex flex-col">
              {/* Header */}
              <div className="flex items-center justify-between border-b px-3 py-2">
                <span className="font-semibold text-foreground">
                  {t("tags.applyTitle")}
                </span>
                <PopoverClose
                  render={
                    <button
                      type="button"
                      className="grid size-5 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                      aria-label={t("actions.close")}
                    />
                  }
                >
                  <X className="size-3.5" />
                </PopoverClose>
              </div>

              {/* Filter Input */}
              <div className="border-b p-2">
                <div className="relative">
                  <Search
                    className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
                    aria-hidden="true"
                  />
                  <Input
                    autoFocus
                    value={filterQuery}
                    placeholder={t("tags.filterPlaceholder")}
                    className="h-8 pl-8 pr-7 text-xs"
                    onChange={(e) => setFilterQuery(e.target.value)}
                    onKeyDown={(e) => {
                      if (
                        e.key === "Enter" &&
                        !hasExactMatch &&
                        filterQuery.trim()
                      ) {
                        e.preventDefault();
                        openCreateView(filterQuery.trim());
                      }
                    }}
                  />
                  {filterQuery ? (
                    <button
                      type="button"
                      aria-label={t("tags.clearFilter")}
                      className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                      onClick={() => setFilterQuery("")}
                    >
                      <X className="size-3.5" />
                    </button>
                  ) : null}
                </div>
              </div>

              {/* Tag List */}
              <div className="max-h-60 overflow-y-auto p-1">
                {filteredOptions.length > 0 ? (
                  <div className="space-y-0.5" role="listbox">
                    {filteredOptions.map((tag) => {
                      const isSelected = selectedNames.has(
                        tag.name.toLocaleLowerCase(),
                      );
                      return (
                        <div
                          key={tag.name.toLocaleLowerCase()}
                          role="option"
                          aria-selected={isSelected}
                          tabIndex={0}
                          className={cn(
                            "group flex cursor-pointer items-start gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-muted/70 focus-visible:bg-muted/70 focus-visible:outline-none",
                            isSelected && "bg-muted/40",
                          )}
                          onClick={() => toggleTag(tag)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.preventDefault();
                              toggleTag(tag);
                            }
                          }}
                        >
                          <span className="mt-0.5 flex size-3.5 shrink-0 items-center justify-center">
                            {isSelected ? (
                              <Check
                                className="size-3.5 text-primary"
                                aria-hidden="true"
                              />
                            ) : null}
                          </span>
                          <span
                            className="mt-1 size-2 shrink-0 rounded-full"
                            style={{ backgroundColor: tag.color }}
                            aria-hidden="true"
                          />
                          <div className="min-w-0 flex-1">
                            <span className="block truncate font-medium text-foreground">
                              {tag.name}
                            </span>
                            {tag.description ? (
                              <span className="block truncate text-[11px] text-muted-foreground">
                                {tag.description}
                              </span>
                            ) : null}
                          </div>
                          <button
                            type="button"
                            title={t("tags.edit")}
                            aria-label={t("tags.editTag")}
                            className="grid size-6 shrink-0 place-items-center rounded-md text-muted-foreground opacity-0 transition-opacity hover:bg-background hover:text-foreground group-hover:opacity-100 focus-visible:opacity-100"
                            onClick={(e) => openEditView(tag, e)}
                          >
                            <Pencil className="size-3" />
                          </button>
                        </div>
                      );
                    })}
                  </div>
                ) : null}

                {/* If no exact match and query entered, offer quick create */}
                {!hasExactMatch && filterQuery.trim() ? (
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 rounded-md px-2 py-2 text-left font-medium text-primary transition-colors hover:bg-primary/10"
                    onClick={() => openCreateView(filterQuery.trim())}
                  >
                    <Plus className="size-3.5 shrink-0" aria-hidden="true" />
                    <span className="truncate">
                      {t("tags.createTagQuery", {
                        name: filterQuery.trim(),
                      })}
                    </span>
                  </button>
                ) : filteredOptions.length === 0 ? (
                  <div className="py-6 text-center text-xs text-muted-foreground">
                    {t("tags.noMatching")}
                  </div>
                ) : null}
              </div>

              {/* Footer: Create new tag button */}
              <div className="border-t p-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-7 w-full gap-1.5 text-xs font-normal"
                  onClick={() => openCreateView()}
                >
                  <Plus className="size-3.5" aria-hidden="true" />
                  {t("tags.createNew")}
                </Button>
              </div>
            </div>
          ) : (
            /* Create / Edit Tag Sub-view */
            <div className="flex flex-col">
              {/* Header with Back button */}
              <div className="flex items-center justify-between border-b px-3 py-2">
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    className="grid size-5 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                    aria-label={t("tags.back")}
                    onClick={() => setView("list")}
                  >
                    <ArrowLeft className="size-3.5" />
                  </button>
                  <span className="font-semibold text-foreground">
                    {view === "edit" ? t("tags.editTag") : t("tags.createNew")}
                  </span>
                </div>
                <PopoverClose
                  render={
                    <button
                      type="button"
                      className="grid size-5 place-items-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                      aria-label={t("actions.close")}
                    />
                  }
                >
                  <X className="size-3.5" />
                </PopoverClose>
              </div>

              {/* Form Body */}
              <div className="space-y-3 p-3">
                {/* Live Preview */}
                <div className="flex items-center gap-2 rounded-md border bg-muted/20 p-2">
                  <span className="text-[11px] text-muted-foreground">
                    {t("tags.preview")}:
                  </span>
                  <AssetTagBadge tag={previewTag} />
                </div>

                {/* Tag Name Input */}
                <div className="space-y-1">
                  <label
                    htmlFor={`${inputId}-tag-name`}
                    className="text-[11px] font-medium text-muted-foreground"
                  >
                    {t("tags.tagName")}
                  </label>
                  <Input
                    id={`${inputId}-tag-name`}
                    autoFocus
                    value={formName}
                    maxLength={64}
                    placeholder={t("tags.tagNamePlaceholder")}
                    aria-label={t("tags.customLabel")}
                    className="h-8 text-xs"
                    onChange={(e) => setFormName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && formName.trim()) {
                        e.preventDefault();
                        handleSaveForm();
                      }
                    }}
                  />
                </div>

                {/* Description Input */}
                <div className="space-y-1">
                  <label
                    htmlFor={`${inputId}-tag-desc`}
                    className="text-[11px] font-medium text-muted-foreground"
                  >
                    {t("tags.tagDescription")}
                  </label>
                  <Input
                    id={`${inputId}-tag-desc`}
                    value={formDescription}
                    maxLength={128}
                    placeholder={t("tags.tagDescriptionPlaceholder")}
                    className="h-8 text-xs"
                    onChange={(e) => setFormDescription(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && formName.trim()) {
                        e.preventDefault();
                        handleSaveForm();
                      }
                    }}
                  />
                </div>

                {/* Color Picker & Presets */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <span className="text-[11px] font-medium text-muted-foreground">
                      {t("tags.tagColor")}
                    </span>
                    <button
                      type="button"
                      className="flex items-center gap-1 text-[11px] text-muted-foreground transition-colors hover:text-foreground"
                      onClick={() =>
                        setFormColor(getRandomAssetTagColor(formColor))
                      }
                    >
                      <Dices className="size-3" aria-hidden="true" />
                      {t("tags.randomColor")}
                    </button>
                  </div>

                  {/* Preset Swatches */}
                  <div className="grid grid-cols-5 gap-1.5">
                    {presetAssetTagColors.map((color) => {
                      const isChosen =
                        formColor.toLowerCase() === color.toLowerCase();
                      return (
                        <button
                          key={color}
                          type="button"
                          className={cn(
                            "relative flex h-6 w-full items-center justify-center rounded-md border transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                            isChosen && "ring-2 ring-foreground/40",
                          )}
                          style={{ backgroundColor: color }}
                          onClick={() => setFormColor(color)}
                          aria-label={color}
                        >
                          {isChosen ? (
                            <Check
                              className="size-3 text-white drop-shadow-xs"
                              aria-hidden="true"
                            />
                          ) : null}
                        </button>
                      );
                    })}
                  </div>

                  {/* Custom Hex + Color swatch trigger */}
                  <div className="flex items-center gap-2 pt-1">
                    <Popover
                      open={showColorPicker}
                      onOpenChange={setShowColorPicker}
                    >
                      <PopoverTrigger
                        render={
                          <button
                            type="button"
                            className="size-7 shrink-0 rounded-md border transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            style={{ backgroundColor: formColor }}
                            aria-label="Toggle custom color picker"
                          />
                        }
                      />
                      <PopoverContent
                        side="left"
                        align="start"
                        sideOffset={8}
                        className="w-64 p-3"
                      >
                        <HexColorPicker
                          color={
                            /^#[0-9a-f]{6}$/i.test(formColor)
                              ? formColor
                              : defaultAssetTagColor
                          }
                          onChange={(c) => setFormColor(c.toUpperCase())}
                          className="!h-40 !w-full rounded-md"
                        />
                      </PopoverContent>
                    </Popover>
                    <Input
                      value={formColor}
                      maxLength={7}
                      className="h-7 text-xs font-mono uppercase"
                      onChange={(e) => setFormColor(e.target.value)}
                    />
                  </div>
                </div>
              </div>

              {/* Form Footer */}
              <div className="flex items-center justify-end gap-2 border-t p-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setView("list")}
                >
                  {t("tags.cancel")}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  disabled={!formName.trim()}
                  className="h-7 text-xs"
                  onClick={handleSaveForm}
                >
                  {view === "edit" ? t("tags.save") : t("tags.create")}
                </Button>
              </div>
            </div>
          )}
        </PopoverContent>

        {/* Applied Tags Container */}
        <div
          id={inputId}
          role="group"
          aria-labelledby={`${inputId}-label`}
          className="flex min-h-12 flex-wrap items-center gap-1.5 rounded-lg border bg-muted/20 p-2.5"
        >
          {tags.length > 0 ? (
            tags.map((tag) => (
              <AssetTagBadge
                key={tag.name.toLocaleLowerCase()}
                tag={tag}
                disabled={disabled}
                onRemove={() => removeTag(tag)}
              />
            ))
          ) : (
            <span className="text-xs text-muted-foreground">
              {t("tags.empty")}
            </span>
          )}

          <PopoverTrigger
            render={
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={disabled}
                className="h-6 rounded-full border-dashed px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
              />
            }
          >
            <Plus className="size-3" aria-hidden="true" />
            {t("tags.add")}
          </PopoverTrigger>
        </div>

        <p className="text-[11px] leading-4 text-muted-foreground">
          {t("tags.nexusHint")}
        </p>
      </div>
    </Popover>
  );
}
