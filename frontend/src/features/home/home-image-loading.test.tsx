// @vitest-environment happy-dom

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { HomeCapabilities } from "./home-capabilities";
import { HomeHero } from "./home-hero";
import { HomeProjectStory } from "./home-project-story";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: React.ReactNode }) => <a>{children}</a>,
}));

describe("home image loading", () => {
  it("prioritizes the above-the-fold hero image", () => {
    const { container } = render(<HomeHero />);
    const image = container.querySelector(
      'img[src="/home/hero-screenshot.png"]',
    );

    expect(image).not.toBeNull();
    expect(image?.getAttribute("loading")).toBe("eager");
    expect(image?.getAttribute("fetchpriority")).toBe("high");
    expect(image?.getAttribute("decoding")).toBe("async");
  });

  it("defers capability images below the fold", () => {
    const { container } = render(<HomeCapabilities />);
    const images = [...container.querySelectorAll("img")];

    expect(images.length).toBeGreaterThan(0);
    images.forEach((image) => {
      expect(image.getAttribute("loading")).toBe("lazy");
      expect(image.getAttribute("decoding")).toBe("async");
      expect(image.getAttribute("fetchpriority")).toBe("low");
    });
  });

  it("links each documented asset capability to its document section", () => {
    const { container } = render(<HomeCapabilities />);
    const links = [...container.querySelectorAll('a[href^="/docs/"]')];

    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/docs/asset-concepts#character",
      "/docs/asset-concepts#object",
      "/docs/asset-concepts#scenery",
      "/docs/asset-concepts#tileset",
      "/docs/asset-concepts#ui-set",
    ]);
  });

  it("defers the interactive project scene layers", () => {
    const { container } = render(<HomeProjectStory />);

    const images = [...container.querySelectorAll("img")];

    expect(images.length).toBeGreaterThan(0);
    images.forEach((image) => {
      expect(image.getAttribute("loading")).toBe("lazy");
      expect(image.getAttribute("decoding")).toBe("async");
      expect(image.getAttribute("fetchpriority")).toBe("low");
    });
  });
});
