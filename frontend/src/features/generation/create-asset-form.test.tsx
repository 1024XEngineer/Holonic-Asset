// @vitest-environment happy-dom

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";
import { CreateAssetForm } from "./create-asset-form";

afterEach(cleanup);

describe("CreateAssetForm", () => {
  it("keeps a UI Set draft stable after adding a component", () => {
    render(
      withI18n(
        <CreateAssetForm kind="uiset" onCancel={vi.fn()} onCreate={vi.fn()} />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Add component" }));

    expect(screen.getAllByRole("region")).toHaveLength(2);
  });

  it("updates a UI Set canvas dimension", () => {
    render(
      withI18n(
        <CreateAssetForm kind="uiset" onCancel={vi.fn()} onCreate={vi.fn()} />,
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "Canvas width" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "1280 px" }));

    expect(
      screen.getByRole("button", { name: "Canvas width" }).textContent,
    ).toBe("1280 px");
  });

  it("submits a valid UI Set without creating a generation request", async () => {
    const onCreate = vi.fn();
    render(
      withI18n(
        <CreateAssetForm kind="uiset" onCancel={vi.fn()} onCreate={onCreate} />,
      ),
    );

    fireEvent.change(screen.getByLabelText("Asset name"), {
      target: { value: "HUD" },
    });
    fireEvent.change(screen.getByLabelText("Creative brief"), {
      target: { value: "Game HUD" },
    });
    fireEvent.change(screen.getByLabelText("Style"), {
      target: { value: "Pixel art" },
    });
    fireEvent.change(screen.getByLabelText("Component name"), {
      target: { value: "Health bar" },
    });
    fireEvent.change(screen.getByLabelText("Component description"), {
      target: { value: "Shows player health" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create UI Set" }));

    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toBe(
        "UI Set draft is ready. Generation will be connected in a follow-up change.",
      ),
    );
    expect(onCreate).not.toHaveBeenCalled();
  });
});
