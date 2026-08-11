import { describe, expect, it, vi } from "vitest";

import type { QuickGenerationAsset } from "@/model/generation";

import { createQuickGenerationSession } from "./quick-generation-session";

function asset(id: string): QuickGenerationAsset {
  return { id, prompt: `${id} prompt`, size: "64 × 64 px" };
}

function setup() {
  let nextPreview = 0;
  const adapter = {
    createPreviewUrl: vi.fn(() => `blob:reference-${++nextPreview}`),
    revokePreviewUrl: vi.fn(),
  };
  return { adapter, session: createQuickGenerationSession(adapter) };
}

describe("quick generation session", () => {
  it("synchronizes, selects, and resets assets while notifying subscribers", () => {
    const { session } = setup();
    const listener = vi.fn();
    const unsubscribe = session.subscribe(listener);

    session.synchronize([asset("first"), asset("second")]);
    expect(session.getSnapshot().currentAssetId).toBe("first");

    session.selectAsset(asset("second"));
    expect(session.getSnapshot().currentAssetId).toBe("second");

    session.synchronize([asset("first")]);
    expect(session.getSnapshot().currentAssetId).toBe("first");

    session.startNewAsset();
    expect(session.getSnapshot().currentAssetId).toBeNull();
    unsubscribe();
    session.updateDraft({ prompt: "No notification" });
    expect(listener).toHaveBeenCalledTimes(4);
  });

  it("accepts only image references and releases replaced previews", () => {
    const { adapter, session } = setup();

    session.chooseReference(
      new File(["text"], "notes.txt", { type: "text/plain" }),
    );
    expect(adapter.createPreviewUrl).not.toHaveBeenCalled();

    session.chooseReference(
      new File(["first"], "first.png", { type: "image/png" }),
    );
    session.chooseReference(
      new File(["second"], "second.webp", { type: "image/webp" }),
    );
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith("blob:reference-1");
    expect(session.getSnapshot().draft).toMatchObject({
      reference: "blob:reference-2",
      referenceFileName: "second.webp",
    });

    session.clearReference();
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith("blob:reference-2");
    expect(session.getSnapshot().draft.reference).toBe("");
  });

  it("prepares a generation once and applies the completed asset", () => {
    const { session } = setup();
    session.updateDraft({ prompt: "  New hero  ", size: " 128 × 128 px " });
    session.chooseReference(
      new File(["hero"], "hero.png", { type: "image/png" }),
    );

    const submission = session.prepareGeneration();
    expect(submission?.input).toEqual({
      assetId: undefined,
      prompt: "New hero",
      size: "128 × 128 px",
      referenceFileName: "hero.png",
    });

    submission?.complete(asset("generated"));
    submission?.complete(asset("ignored"));
    submission?.fail();
    expect(session.getSnapshot().currentAssetId).toBe("generated");

    session.selectAsset(asset("generated"));
    expect(session.getSnapshot().draft.reference).toBe("blob:reference-1");
  });

  it("handles failed submissions, invalid drafts, and deletion races", () => {
    const { adapter, session } = setup();
    expect(session.prepareGeneration()).toBeUndefined();
    expect(session.prepareDeletion([])).toBeUndefined();

    session.synchronize([asset("first"), asset("second")]);
    expect(session.prepareDeletion([asset("second")])).toBeUndefined();

    session.updateDraft({ prompt: "Variation" });
    session.chooseReference(
      new File(["reference"], "reference.png", { type: "image/png" }),
    );
    const submission = session.prepareGeneration();
    session.clearReference();
    submission?.fail();
    submission?.fail();
    expect(adapter.revokePreviewUrl).toHaveBeenCalledWith("blob:reference-1");

    const deletion = session.prepareDeletion([asset("first"), asset("second")]);
    expect(deletion?.assetId).toBe("first");
    deletion?.complete();
    deletion?.complete();
    expect(session.getSnapshot().currentAssetId).toBe("second");

    const staleDeletion = session.prepareDeletion([asset("second")]);
    session.startNewAsset();
    staleDeletion?.complete();
    session.dispose();
  });
});
