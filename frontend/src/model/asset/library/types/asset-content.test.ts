import { describe, expect, expectTypeOf, it } from "vitest";

import type { Perspective } from "@/model/project";
import type { AssetDetailResponse } from "../asset.contract";
import type { AssetContentByType } from "./asset-content";

type DirectionalAssetType = "character" | "object";

type DirectionalContent<
  Type extends DirectionalAssetType,
  View extends Perspective,
> = NonNullable<
  Extract<AssetDetailResponse, { type: Type; perspective: View }>["content"]
>;

describe("asset content contracts", () => {
  it("preserves top-level metadata for every asset type", () => {
    const content = {
      character: {
        directionCount: 2,
        prototype: [],
        metadata: { source: "generator" },
      },
      object: {
        directionCount: 2,
        prototype: [],
        metadata: { source: "generator" },
      },
      tileSet: { metadata: { source: "generator" } },
      audio: { metadata: { source: "generator" } },
      uiset: { metadata: { source: "generator" } },
      scenery: { metadata: { source: "generator" } },
    } satisfies AssetContentByType<"Side-On">;

    expect(content.scenery.metadata).toEqual({ source: "generator" });
  });

  it("binds directional asset content to the backend perspective mapping", () => {
    expectTypeOf<
      DirectionalContent<"character", "Side-On">["directionCount"]
    >().toEqualTypeOf<2>();
    expectTypeOf<
      DirectionalContent<"character", "Top-Down">["directionCount"]
    >().toEqualTypeOf<4>();
    expectTypeOf<
      DirectionalContent<"character", "Isometric">["directionCount"]
    >().toEqualTypeOf<8>();
    expectTypeOf<
      DirectionalContent<"object", "Side-On">["directionCount"]
    >().toEqualTypeOf<2>();
    expectTypeOf<
      DirectionalContent<"object", "Top-Down">["directionCount"]
    >().toEqualTypeOf<4>();
    expectTypeOf<
      DirectionalContent<"object", "Isometric">["directionCount"]
    >().toEqualTypeOf<8>();
  });
});
