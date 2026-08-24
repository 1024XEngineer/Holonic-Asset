export type NearSquareGrid = {
  columns: number;
  rows: number;
};

export function getNearSquareGrid(itemCount: number): NearSquareGrid {
  const count = Math.max(0, Math.floor(itemCount));
  if (count === 0) return { columns: 0, rows: 0 };

  const columns = Math.ceil(Math.sqrt(count));
  return { columns, rows: Math.ceil(count / columns) };
}
