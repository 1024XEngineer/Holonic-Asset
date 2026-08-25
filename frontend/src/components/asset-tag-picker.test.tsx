// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { AssetTagPicker } from "./asset-tag-picker";

describe("AssetTagPicker", () => {
  afterEach(cleanup);

  it("creates a reusable structured tag via creation sub-view", () => {
    const onChange = vi.fn();
    render(
      withI18n(
        <AssetTagPicker id="asset-tags" tags={[]} onChange={onChange} />,
      ),
    );

    // Open popover by clicking Add tag button
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));

    // Click "Create new tag" in popover footer
    fireEvent.click(screen.getByRole("button", { name: "Create new tag" }));

    // Type tag name
    fireEvent.change(screen.getByLabelText("Tag name"), {
      target: { value: "  village  " },
    });

    // Type description
    fireEvent.change(screen.getByPlaceholderText("Optional description..."), {
      target: { value: "A quiet village" },
    });

    // Pick a preset color (e.g. GitHub Green #1A7F37)
    fireEvent.click(screen.getByLabelText("#1A7F37"));

    // Click submit
    fireEvent.click(screen.getByRole("button", { name: "Create tag" }));

    expect(onChange).toHaveBeenCalledWith([
      { name: "village", description: "A quiet village", color: "#1A7F37" },
    ]);
  });

  it("opens the detailed color picker without resizing the tag popover", () => {
    render(
      withI18n(<AssetTagPicker id="asset-tags" tags={[]} onChange={vi.fn()} />),
    );

    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));
    fireEvent.click(screen.getByRole("button", { name: "Create new tag" }));

    const tagPopover = document.querySelector<HTMLElement>(
      '[data-slot="popover-content"]',
    );
    expect(tagPopover).not.toBeNull();

    fireEvent.click(
      screen.getByRole("button", { name: "Toggle custom color picker" }),
    );

    const colorPicker = document.querySelector<HTMLElement>(".react-colorful");
    const popovers = document.querySelectorAll('[data-slot="popover-content"]');

    expect(colorPicker).not.toBeNull();
    expect(popovers).toHaveLength(2);
    expect(tagPopover?.contains(colorPicker)).toBe(false);
  });

  it("preserves metadata when selecting an existing project tag from list", () => {
    const onChange = vi.fn();
    render(
      withI18n(
        <AssetTagPicker
          availableTags={[
            {
              name: "village",
              description: "Village buildings and residents",
              color: "#F59E0B",
            },
          ]}
          id="asset-tags"
          tags={[]}
          onChange={onChange}
        />,
      ),
    );

    // Open popover
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));

    // Select the "village" option from the list
    fireEvent.click(screen.getByText("village"));

    expect(onChange).toHaveBeenCalledWith([
      {
        name: "village",
        description: "Village buildings and residents",
        color: "#F59E0B",
      },
    ]);
  });

  it("toggles off a selected tag when clicked in popover list", () => {
    const onChange = vi.fn();
    const tag = {
      name: "village",
      description: "Village buildings",
      color: "#0969DA",
    };
    render(
      withI18n(
        <AssetTagPicker
          availableTags={[tag]}
          id="asset-tags"
          tags={[tag]}
          onChange={onChange}
        />,
      ),
    );

    // Open popover
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));

    // Click the selected option to toggle it off
    fireEvent.click(screen.getByRole("option", { name: /village/i }));

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("removes a tag directly from the badge remove button", () => {
    const onChange = vi.fn();
    const tag = {
      name: "village",
      description: "Village buildings",
      color: "#0969DA",
    };
    render(
      withI18n(
        <AssetTagPicker id="asset-tags" tags={[tag]} onChange={onChange} />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Remove village" }));

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("filters tags by query and offers quick create when not found", () => {
    const onChange = vi.fn();
    render(
      withI18n(
        <AssetTagPicker
          availableTags={[
            { name: "forest", description: "Trees", color: "#1A7F37" },
          ]}
          id="asset-tags"
          tags={[]}
          onChange={onChange}
        />,
      ),
    );

    // Open popover
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));

    // Type filter query
    fireEvent.change(screen.getByPlaceholderText("Filter tags..."), {
      target: { value: "castle" },
    });

    // Quick create button appears
    const quickCreateBtn = screen.getByRole("button", {
      name: /Create tag "castle"/i,
    });
    fireEvent.click(quickCreateBtn);

    // Now in create view, formName is pre-filled with "castle"
    expect((screen.getByLabelText("Tag name") as HTMLInputElement).value).toBe(
      "castle",
    );

    // Click Create tag
    fireEvent.click(screen.getByRole("button", { name: "Create tag" }));

    expect(onChange).toHaveBeenCalledWith([
      expect.objectContaining({ name: "castle" }),
    ]);
  });

  it("edits an existing tag via the edit sub-view", () => {
    const onChange = vi.fn();
    const tag = {
      name: "village",
      description: "Old description",
      color: "#0969DA",
    };
    render(
      withI18n(
        <AssetTagPicker
          availableTags={[tag]}
          id="asset-tags"
          tags={[tag]}
          onChange={onChange}
        />,
      ),
    );

    // Open popover
    fireEvent.click(screen.getByRole("button", { name: "Add tag" }));

    // Click edit pencil button
    fireEvent.click(screen.getByRole("button", { name: "Edit tag" }));

    // Change description
    fireEvent.change(screen.getByPlaceholderText("Optional description..."), {
      target: { value: "Updated description" },
    });

    // Click Save changes
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    expect(onChange).toHaveBeenCalledWith([
      {
        name: "village",
        description: "Updated description",
        color: "#0969DA",
      },
    ]);
  });
});
