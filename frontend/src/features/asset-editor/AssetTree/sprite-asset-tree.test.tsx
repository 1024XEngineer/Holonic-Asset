// @vitest-environment happy-dom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { withI18n } from "@/testing/with-i18n";

import { SpriteAssetTree } from "./sprite-asset-tree";

afterEach(cleanup);

const animation = {
  kind: "clip" as const,
  id: "7",
  label: "Walk",
  frameCount: 4,
  generation: {
    direction: "left",
    frameCount: 4,
    columns: 4,
    frameWidth: 48,
    frameHeight: 48,
    fps: 12,
    resolution: "48x48",
    duration: 5,
    aspectRatio: "1:1",
  },
};

describe("SpriteAssetTree", () => {
  it("derives only the missing directions of an animation group", () => {
    const onDeriveAnimation = vi.fn();
    render(
      withI18n(
        <SpriteAssetTree
          animations={[
            animation,
            {
              ...animation,
              id: "8",
              label: "Walk right",
              groupId: "7",
              generation: { ...animation.generation, direction: "right" },
            },
          ]}
          perspective="Top-Down"
          selectedNode={null}
          selectedFrames={[]}
          onSelect={vi.fn()}
          onSelectFrame={vi.fn()}
          onGenerateAnimation={vi.fn()}
          onDeriveAnimation={onDeriveAnimation}
          onRenameAnimation={vi.fn()}
          onDeleteAnimation={vi.fn()}
          isGeneratingAnimation={false}
        />,
      ),
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Derive directions for Walk" }),
    );

    expect(screen.getByRole("button", { name: "Front" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Back" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Right" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Left" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Front" }));
    fireEvent.click(screen.getByRole("button", { name: "Front" }));
    fireEvent.click(screen.getByRole("button", { name: "Front" }));
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Generate directions" }),
    );

    expect(onDeriveAnimation).toHaveBeenCalledWith({
      sourceAnimationId: "7",
      sourceAnimationName: "Walk",
      targetDirections: ["front", "back"],
    });
  });

  it("disables derivation for animations without generation metadata", () => {
    render(
      withI18n(
        <SpriteAssetTree
          animations={[{ ...animation, generation: undefined }]}
          perspective="Top-Down"
          selectedNode={null}
          selectedFrames={[]}
          onSelect={vi.fn()}
          onSelectFrame={vi.fn()}
          onGenerateAnimation={vi.fn()}
          onDeriveAnimation={vi.fn()}
          onRenameAnimation={vi.fn()}
          onDeleteAnimation={vi.fn()}
          isGeneratingAnimation={false}
        />,
      ),
    );

    expect(
      (
        screen.getByRole("button", {
          name: "Derive directions for Walk",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
  });
});
