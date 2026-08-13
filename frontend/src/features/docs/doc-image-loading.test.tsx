// @vitest-environment happy-dom

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DocImage } from "./doc-text";
import { TwoDirectionExample } from "./direction-examples";

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

  it("prioritizes only the first two-direction example image", () => {
    const { container } = render(<TwoDirectionExample priority />);
    const firstImage = container.querySelector(
      'img[src="/assets/characters/swordsman/idle-left.png"]',
    );
    const secondImage = container.querySelector(
      'img[src="/assets/characters/swordsman/idle-right.png"]',
    );

    expect(firstImage?.getAttribute("loading")).toBe("eager");
    expect(firstImage?.getAttribute("fetchpriority")).toBe("high");
    expect(secondImage?.getAttribute("loading")).toBe("lazy");
    expect(secondImage?.getAttribute("fetchpriority")).toBe("low");
  });
});
