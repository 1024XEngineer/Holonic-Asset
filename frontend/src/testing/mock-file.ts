export function createMockFile(
  name = "reference.png",
  contentType = "image/png",
) {
  return new File(["mock asset content"], name, { type: contentType });
}
