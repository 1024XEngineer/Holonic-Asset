// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { CreateTilesetItemTrigger } from "./create-tileset-item-trigger";

afterEach(cleanup);

describe("CreateTilesetItemTrigger", () => {
  it("does not submit an incomplete request", () => {
    const onGenerate = vi.fn();
    render(
      withI18n(
        <CreateTilesetItemTrigger isGenerating={false} onGenerate={onGenerate}>
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateTilesetItemTrigger>,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Open form" }));
    fireEvent.submit(document.body.querySelector("form")!);

    expect(onGenerate).not.toHaveBeenCalled();
  });

  it("shows the queueing state while generation is in progress", () => {
    render(
      withI18n(
        <CreateTilesetItemTrigger isGenerating onGenerate={vi.fn()}>
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateTilesetItemTrigger>,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Open form" }));

    expect(screen.getByText("Adding item to queue...")).toBeTruthy();
  });

  it("collects item metadata, footprint, and creative brief", async () => {
    const user = userEvent.setup();
    const onGenerate = vi.fn();
    render(
      withI18n(
        <CreateTilesetItemTrigger isGenerating={false} onGenerate={onGenerate}>
          {(openDialog) => (
            <button type="button" onClick={openDialog}>
              Open form
            </button>
          )}
        </CreateTilesetItemTrigger>,
      ),
    );

    await user.click(screen.getByRole("button", { name: "Open form" }));
    await user.type(screen.getByLabelText("Item 1 name"), "Oak tree");
    await user.type(screen.getByLabelText("Item 1 description"), "Old oak");
    await user.click(
      screen.getByRole("button", { name: "Tile column 2, row 1" }),
    );
    await user.type(
      screen.getByLabelText("Creative brief"),
      "Dense leaves and warm bark",
    );
    await user.click(screen.getByRole("button", { name: "Add tileset item" }));

    expect(onGenerate).toHaveBeenCalledWith({
      itemName: "Oak tree",
      itemDescription: "Old oak",
      shape: [
        [0, 0],
        [1, 0],
      ],
      creativeBrief: "Dense leaves and warm bark",
    });
  });
});
