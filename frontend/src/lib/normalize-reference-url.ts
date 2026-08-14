const qiniuS3HostSuffix = ".qiniucs.com";
const s3SignatureParameters = [
  "X-Amz-Algorithm",
  "X-Amz-Credential",
  "X-Amz-Date",
  "X-Amz-Expires",
  "X-Amz-Signature",
  "X-Amz-SignedHeaders",
] as const;

/**
 * Converts a Qiniu S3 presigned URL into its stable object URL before it is
 * persisted or sent to the API. The API can then resolve a fresh URL later.
 */
export function normalizeReferenceUrl(reference: string) {
  const value = reference.trim();
  if (!/^https?:\/\//i.test(value)) return value;

  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    return value;
  }

  if (!isQiniuS3Host(parsed.hostname) || !hasS3Signature(parsed)) {
    return value;
  }

  parsed.search = "";
  parsed.hash = "";
  return parsed.toString();
}

function isQiniuS3Host(hostname: string) {
  const host = hostname.toLowerCase();
  return (
    host === qiniuS3HostSuffix.slice(1) || host.endsWith(qiniuS3HostSuffix)
  );
}

function hasS3Signature(url: URL) {
  return s3SignatureParameters.every((parameter) =>
    url.searchParams.has(parameter),
  );
}
