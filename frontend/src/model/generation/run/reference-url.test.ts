import { describe, expect, it } from "vitest";

import { normalizeReferenceUrl } from "@/lib/normalize-reference-url";

describe("normalizeReferenceUrl", () => {
  it("removes temporary AWS signature parameters from Qiniu S3 URLs", () => {
    const reference =
      "https://xe-6-2.s3.cn-east-1.qiniucs.com/uploads/reference.png" +
      "?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
      "&X-Amz-Credential=access%2Fscope" +
      "&X-Amz-Date=20260814T063100Z" +
      "&X-Amz-Expires=1800" +
      "&X-Amz-SignedHeaders=host" +
      "&X-Amz-Signature=signature";

    expect(normalizeReferenceUrl(reference)).toBe(
      "https://xe-6-2.s3.cn-east-1.qiniucs.com/uploads/reference.png",
    );
  });

  it("preserves ordinary URLs, data URLs, and signatures from other hosts", () => {
    const signedOtherHost =
      "https://cdn.example.com/reference.png?X-Amz-Signature=signature";

    expect(normalizeReferenceUrl("data:image/png;base64,abc")).toBe(
      "data:image/png;base64,abc",
    );
    expect(normalizeReferenceUrl("https://cdn.example.com/reference.png")).toBe(
      "https://cdn.example.com/reference.png",
    );
    expect(normalizeReferenceUrl(signedOtherHost)).toBe(signedOtherHost);
  });

  it("does not strip non-signature query parameters from Qiniu URLs", () => {
    const reference = "https://cdn.qiniucs.com/reference.png?version=2";

    expect(normalizeReferenceUrl(reference)).toBe(reference);
  });
});
