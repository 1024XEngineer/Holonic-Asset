// @vitest-environment happy-dom

import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { createAssetCreationDraft } from "../lib";
import type { UISetAssetCreationDraft } from "../types";
import { UISetAssetFields } from "./uiset-asset-fields";

function UISetFieldsTestHarness() {
  const [draft, setDraft] = useState<UISetAssetCreationDraft<File>>(
    () =>
      createAssetCreationDraft<File>("uiset") as UISetAssetCreationDraft<File>,
  );

  return <UISetAssetFields draft={draft} onChange={setDraft} />;
}

describe("UISetAssetFields", () => {
  it("preserves collapsed components when adding a component", () => {
    render(withI18n(<UISetFieldsTestHarness />));

    fireEvent.click(
      screen.getByRole("button", { name: "Collapse component 1" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add component" }));

    expect(
      screen.getByRole("button", { name: "Expand component 1" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Collapse component 2" }),
    ).toBeTruthy();
  });
});
