export function getGridRowCount(itemCount: number, columnCount: number) {
  return Math.ceil(Math.max(1, itemCount) / Math.max(1, columnCount));
}
