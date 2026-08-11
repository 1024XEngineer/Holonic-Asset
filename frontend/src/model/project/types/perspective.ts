import type { components } from "@/model/generated/core-api";
import { z } from "zod";

type CorePerspective = components["schemas"]["ProjectResponse"]["perspective"];

const perspectiveValues = [
  "Top-Down",
  "Side-On",
  "Isometric",
] as const satisfies readonly CorePerspective[];

export const perspectiveSchema = z.enum(perspectiveValues);

export type Perspective = z.infer<typeof perspectiveSchema>;

export const perspectiveOptions = Object.freeze([...perspectiveValues]);

export function isPerspective(value: unknown): value is Perspective {
  return perspectiveSchema.safeParse(value).success;
}
