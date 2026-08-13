// @vitest-environment happy-dom

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DocImage } from "./doc-text";

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

  it("prioritizes the first image in an article when requested", () => {
    const { container } = render(
      <DocImage src="/example.png" altKey="reference.heading" priority />,
    );
    const image = container.querySelector("img");

    expect(image?.getAttribute("loading")).toBe("eager");
    expect(image?.getAttribute("fetchpriority")).toBe("high");
  });
});
