import type { components } from "@/model/generated/core-api";

type CorePerspective = components["schemas"]["ProjectResponse"]["perspective"];

const perspectiveValues = {
  "Top-Down": true,
  "Side-On": true,
  Isometric: true,
} as const satisfies Record<CorePerspective, true>;

export type Perspective = keyof typeof perspectiveValues;

export const perspectiveOptions = Object.freeze(
  Object.keys(perspectiveValues) as Perspective[],
);

export function isPerspective(value: string): value is Perspective {
  return Object.hasOwn(perspectiveValues, value);
}
