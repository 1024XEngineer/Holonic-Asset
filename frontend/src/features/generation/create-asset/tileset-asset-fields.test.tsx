// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { createAssetCreationDraft } from "../lib";
import type { TilesetAssetCreationDraft } from "../types";
import { TilesetAssetFields } from "./tileset-asset-fields";

afterEach(cleanup);

function TilesetFieldsTestHarness() {
  const [draft, setDraft] = useState<TilesetAssetCreationDraft>(
    () => createAssetCreationDraft("tileset") as TilesetAssetCreationDraft,
  );

  return <TilesetAssetFields draft={draft} onChange={setDraft} />;
}

describe("TilesetAssetFields", () => {
  it("preserves collapsed items while editing another item", () => {
    render(withI18n(<TilesetFieldsTestHarness />));

    fireEvent.click(screen.getByRole("button", { name: "Item count" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "3" }));
    fireEvent.click(screen.getByRole("button", { name: "Collapse item 2" }));

    fireEvent.change(screen.getByLabelText("Item 1 name"), {
      target: { value: "Grass edge" },
    });

    expect(screen.getByRole("button", { name: "Expand item 2" })).toBeTruthy();
  });
});
