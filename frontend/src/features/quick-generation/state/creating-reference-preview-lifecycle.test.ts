import { describe, expect, it, vi } from "vitest";

import { createCreatingReferencePreviewLifecycle } from "./creating-reference-preview-lifecycle";

function setup() {
  let nextId = 0;
  const adapter = {
    createPreviewUrl: vi.fn(() => `blob:preview-${++nextId}`),
    revokePreviewUrl: vi.fn(),
  };
  return {
    adapter,
    lifecycle: createCreatingReferencePreviewLifecycle(adapter),
  };
}

describe("creating reference preview lifecycle", () => {
  it("defers revocation while a preview is retained by a submission", () => {
    const { adapter, lifecycle } = setup();
    const url = lifecycle.create(new File(["image"], "reference.png"));

    lifecycle.retainForSubmission(url);
    lifecycle.releaseUncommitted(url, null);
    expect(adapter.revokePreviewUrl).not.toHaveBeenCalled();

    lifecycle.settleSubmission(url, false, "blob:other");
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith(url);
  });

  it("tracks committed previews and revokes replacements", () => {
    const { adapter, lifecycle } = setup();
    const first = lifecycle.create(new File(["first"], "first.png"));
    const second = lifecycle.create(new File(["second"], "second.png"));

    lifecycle.commit("asset-1", first);
    expect(lifecycle.previewForAsset("asset-1")).toBe(first);
    lifecycle.releaseUncommitted(first, "asset-1");
    expect(adapter.revokePreviewUrl).not.toHaveBeenCalled();

    lifecycle.commit("asset-1", second);
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith(first);
    expect(lifecycle.previewForAsset("asset-1")).toBe(second);

    lifecycle.releaseAsset("asset-1");
    lifecycle.releaseAsset("missing");
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith(second);
    expect(lifecycle.previewForAsset("asset-1")).toBeUndefined();
  });

  it("keeps successful submissions and disposes every remaining URL", () => {
    const { adapter, lifecycle } = setup();
    const kept = lifecycle.create(new File(["kept"], "kept.png"));
    const pending = lifecycle.create(new File(["pending"], "pending.png"));

    lifecycle.retainForSubmission(kept);
    lifecycle.releaseUncommitted(kept, null);
    lifecycle.settleSubmission(kept, true, "");
    lifecycle.commit("asset-1", kept);
    lifecycle.commit("asset-1", "");
    lifecycle.retainForSubmission("https://example.com/external.png");
    lifecycle.settleSubmission("", false, undefined);
    lifecycle.dispose();

    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith(kept);
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith(pending);
    expect(lifecycle.previewForAsset("asset-1")).toBeUndefined();
  });
});
