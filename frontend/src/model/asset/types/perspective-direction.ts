import type { Perspective } from "@/model/project";

export const perspectiveDirectionLayouts = {
  "Top-Down": { directionCount: 4, columns: 2, rows: 2 },
  "Side-On": { directionCount: 2, columns: 2, rows: 1 },
  Isometric: { directionCount: 8, columns: 4, rows: 2 },
} as const satisfies Record<
  Perspective,
  { directionCount: number; columns: number; rows: number }
>;

type DirectionCountByPerspective = {
  [View in Perspective]: (typeof perspectiveDirectionLayouts)[View]["directionCount"];
};

export type DirectionCountForPerspective<
  View extends Perspective = Perspective,
> = DirectionCountByPerspective[View];

export function getPerspectiveDirectionLayout<View extends Perspective>(
  perspective: View,
) {
  return perspectiveDirectionLayouts[perspective];
}
