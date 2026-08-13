// @vitest-environment happy-dom

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DocImage } from "./doc-text";
import { TwoDirectionExample } from "./direction-examples";
import { PerspectiveExample } from "./perspective-examples";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("docs image loading", () => {
  it("defers document images and decodes them asynchronously", () => {
    const { container } = render(
      <DocImage src="/example.png" altKey="reference.heading" />,
    );
    const image = container.querySelector("img");

    expect(image?.getAttribute("loading")).toBe("lazy");
    expect(image?.getAttribute("decoding")).toBe("async");
    expect(image?.getAttribute("fetchpriority")).toBe("low");
  });

  it("applies eager loading and high priority when requested", () => {
    const { container } = render(
      <DocImage src="/example.png" altKey="reference.heading" priority />,
    );
    const image = container.querySelector("img");

    expect(image?.getAttribute("loading")).toBe("eager");
    expect(image?.getAttribute("fetchpriority")).toBe("high");
  });

  it("prioritizes both above-the-fold two-direction images", () => {
    const { container } = render(<TwoDirectionExample priority />);
    const firstImage = container.querySelector(
      'img[src="/assets/characters/swordsman/idle-left.png"]',
    );
    const secondImage = container.querySelector(
      'img[src="/assets/characters/swordsman/idle-right.png"]',
    );

    expect(firstImage?.getAttribute("loading")).toBe("eager");
    expect(firstImage?.getAttribute("fetchpriority")).toBe("high");
    expect(secondImage?.getAttribute("loading")).toBe("eager");
    expect(secondImage?.getAttribute("fetchpriority")).toBe("high");
  });

  it("reserves intrinsic space for perspective images", () => {
    const { container } = render(
      <PerspectiveExample
        image="isometric.png"
        altKey="perspective.isometricAlt"
      />,
    );
    const image = container.querySelector("img");

    expect(image?.getAttribute("width")).toBe("1403");
    expect(image?.getAttribute("height")).toBe("814");
  });
});
