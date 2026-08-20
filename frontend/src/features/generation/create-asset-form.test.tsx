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
  it.each([
    ["audio", "Describe the mood, instruments, rhythm, and intended use..."],
    [
      "scenery",
      "Describe the complete scene, atmosphere, depth, and important elements. Layers will be planned automatically...",
    ],
    [
      "object",
      "Describe the subject, material, mood, and details to generate...",
    ],
  ] as const)(
    "shows the %s creative brief placeholder",
    (kind, placeholder) => {
      render(
        withI18n(
          <CreateAssetForm kind={kind} onCancel={vi.fn()} onCreate={vi.fn()} />,
        ),
      );

      expect(screen.getByLabelText("Creative brief")).toHaveProperty(
        "placeholder",
        placeholder,
      );
    },
  );

  it("submits scenery using only the common creation fields", async () => {
    const onCreate = vi.fn();
    render(
      withI18n(
        <CreateAssetForm
          kind="scenery"
          onCancel={vi.fn()}
          onCreate={onCreate}
        />,
      ),
    );

    fireEvent.change(screen.getByLabelText("Asset name"), {
      target: { value: "Moonlit orchard" },
    });
    fireEvent.change(screen.getByLabelText("Creative brief"), {
      target: { value: "An orchard under a full moon" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create Scenery" }));

    await waitFor(() => expect(onCreate).toHaveBeenCalledOnce());
    expect(onCreate).toHaveBeenCalledWith({
      kind: "scenery",
      name: "Moonlit orchard",
      prompt: "An orchard under a full moon",
      canvasSize: "32 × 32 px",
    });
  });

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
