export type GridCoordinate = readonly [column: number, row: number];

export function getGridBounds(coordinates: readonly GridCoordinate[]) {
  const minX = Math.min(...coordinates.map(([x]) => x));
  const minY = Math.min(...coordinates.map(([, y]) => y));
  const maxX = Math.max(...coordinates.map(([x]) => x));
  const maxY = Math.max(...coordinates.map(([, y]) => y));

  return {
    x: minX,
    y: minY,
    width: maxX - minX + 1,
    height: maxY - minY + 1,
  };
}
