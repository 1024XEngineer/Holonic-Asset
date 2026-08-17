// @vitest-environment happy-dom

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AssetLibraryItem } from "@/model/asset";
import { withI18n } from "@/testing/with-i18n";

import { AssetCard } from "./asset-card";

vi.mock("./asset-preview", () => ({
  AssetPreview: () => <div />,
}));

const readOnlyAsset: AssetLibraryItem = {
  id: "demo-ui-set",
  kind: "uiset",
  name: "Demo UI Set",
  description: "A frontend-only UI Set fixture.",
  version: "v1",
  canvasSize: "1280 × 720 px",
  perspective: "Top-Down",
  tags: ["demo"],
  isReadOnly: true,
  history: [],
  animations: [],
};

describe("AssetCard", () => {
  it("disables Core API actions for read-only assets", () => {
    render(
      withI18n(
        <AssetCard
          asset={readOnlyAsset}
          isCopying={false}
          isDeleting={false}
          onCopy={vi.fn()}
          onDelete={vi.fn()}
          onEdit={vi.fn()}
          onOpenEditor={vi.fn()}
        />,
      ),
    );

    expect(
      screen
        .getByRole("button", { name: "Copy Demo UI Set" })
        .hasAttribute("disabled"),
    ).toBe(true);
    expect(
      screen
        .getByRole("button", { name: "Edit Demo UI Set details" })
        .hasAttribute("disabled"),
    ).toBe(true);
    expect(
      screen
        .getByRole("button", { name: "Delete Demo UI Set" })
        .hasAttribute("disabled"),
    ).toBe(true);
    expect(
      screen
        .getByRole("button", { name: "Open UI Set editor" })
        .hasAttribute("disabled"),
    ).toBe(false);
  });
});
